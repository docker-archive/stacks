package dispatcher

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/errdefs"

	"github.com/docker/stacks/pkg/interfaces"
	"github.com/docker/stacks/pkg/reconciler/notifier"
	"github.com/docker/stacks/pkg/reconciler/reconciler"
)

// Dispatcher is the object that decides when to call the reconciler and with
// what objects. It exists separately from the Reconciler so that we can
// decouple the channel-driven logic of choosing events to reconcile from the
// function-type logic of reconciling.
type Dispatcher interface {
	notifier.ObjectChangeNotifier

	HandleEvents(chan interface{}) error
}

// dispatcher implements the Dispatcher interface
type dispatcher struct {
	mu sync.Mutex

	r reconciler.Reconciler

	// currently, the reconciler package only works with Stacks. The
	// dispatcher will be updated to handle more object types as the
	// Reconciler implements functionality for them.

	// pendingStacks (and the similar pending maps) are sets of object IDs.
	// at first glance, we might want to put all objects into a
	// map[string]*interfaces.ReconcileResource, where the key is the ID
	// and the value is the kind. however, we have to reconcile objects in
	// order: stacks, then networks, configs, and secrets,
	// and finally services.
	pendingStacks   map[string]*interfaces.ReconcileResource
	pendingNetworks map[string]*interfaces.ReconcileResource
	pendingSecrets  map[string]*interfaces.ReconcileResource
	pendingConfigs  map[string]*interfaces.ReconcileResource
	pendingServices map[string]*interfaces.ReconcileResource
}

// New creates and returns the default Dispatcher object, which will
// work on the provided Reconciler
func New(r reconciler.Reconciler, register notifier.Register) Dispatcher {
	return newDispatcher(r, register)
}

// newDispatcher is the private method that creates a new dispatcher object. It
// exists separately for testing purposes.
func newDispatcher(r reconciler.Reconciler, register notifier.Register) *dispatcher {
	m := &dispatcher{
		r:               r,
		pendingStacks:   map[string]*interfaces.ReconcileResource{},
		pendingNetworks: map[string]*interfaces.ReconcileResource{},
		pendingSecrets:  map[string]*interfaces.ReconcileResource{},
		pendingConfigs:  map[string]*interfaces.ReconcileResource{},
		pendingServices: map[string]*interfaces.ReconcileResource{},
	}
	register.Register(m)
	return m
}

// NewRequest creates a new request to reconcile a resource
func NewRequest(kind, ID string) (*interfaces.ReconcileResource, error) {
	reconcileKind := kind
	if _, ok := interfaces.ReconcileKinds[reconcileKind]; !ok {
		return nil, errdefs.NotFound(fmt.Errorf("Resource kind %s not found", kind))
	}

	result := interfaces.ReconcileResource{
		SnapshotResource: interfaces.SnapshotResource{
			ID: ID,
		},
		Kind: reconcileKind,
	}
	return &result, nil
}

// Notify tells the dispatcher to call the reconciler with this object at some
// point in the future
func (d *dispatcher) Notify(request *interfaces.ReconcileResource) {
	id := request.ID
	d.mu.Lock()
	defer d.mu.Unlock()
	switch request.Kind {
	case interfaces.ReconcileStack:
		d.pendingStacks[id] = request
	case interfaces.ReconcileNetwork:
		d.pendingNetworks[id] = request
	case interfaces.ReconcileSecret:
		d.pendingSecrets[id] = request
	case interfaces.ReconcileConfig:
		d.pendingConfigs[id] = request
	case interfaces.ReconcileService:
		d.pendingServices[id] = request
	}
}

// HandleEvents takes a channel that issues events, and processes those events
// by handing them off to the Reconciler. It exits when the provided channel is
// closed. This occurs immediately, and no further calls to the reconciler will
// subsequently be made.
//
// The channel for eventC is nominally of type interface{}, but the returned
// objects must all be of type events.Messages. The odd type of eventC is a
// consequence of the docker daemon Backend API.
//
// HandleEvents will usually deal with errors itself; however, if a
// serious error occurs, it may return an error indicating this.
func (d *dispatcher) HandleEvents(eventC chan interface{}) error {
	// HandleEvents is a state machine. It looks like this:
	//                                           ________
	//                                          / ______ \
	//                                          ||      ||
	//      _         +------------------------>|| exit ||
	//     |_|        |                         ||______||
	//      |         |                         \________/
	//      |         | channel closed              ^
	//      | start   |                             | channel closed
	//  ____V_________|_                      ______|_________
	// |                |   channel read     |                |
	// | wait for read  |------------------->| reading events |<-+
	// |________________|                    |________________|  |
	//         ^                               |   ^   |         | channel read
	//         |               channel blocked |   |   +---------+
	//         |                               |   |
	//         |                               |   | Some objects left
	//         |                    ___________V___|_______
	//         |                   |                       |
	//         +-------------------|  Reconcile one object |
	//           no objects left   |_______________________|
	//

	// the whole thing  goes in a for loop
	for {
		// initial state: waiting for a channel read
		ev, ok := <-eventC
		if !ok {
			// if the channel is closed, return
			return nil
		}
		err := d.resolveMessage(ev)
		if err != nil {
			logrus.Error(err)
		}
		// next state: reading events
	readingEvents:
		for {
			// read as long as the channel is ready
			select {
			case ev, ok := <-eventC:
				// channel closed, return
				if !ok {
					return nil
				}
				err = d.resolveMessage(ev)
				if err != nil {
					logrus.Error(err)
				}
			default:
				// when the channel is no longer ready, process an event
				request := d.pickObject()
				if request == nil {
					// if there are no more objects in the queue, go back to
					// waiting for an event
					break readingEvents
				}
				// next state: reconcile the object. if it fails, add it back
				// to the set of objects.
				err = d.r.Reconcile(request)
				if err != nil {
					// TODO(dperny): if a given object always fails, we'll stay
					// in this state forever, looping again and again.
					logrus.Error(err)
					d.Notify(request)
				}
			}
		}
	}
}

// resolveMessage is a method that figures out what kind of event this is and
// puts it into the correct map
func (d *dispatcher) resolveMessage(ev interface{}) error {
	// naked type cast. If this isn't events.Message, then the program will
	// panic. This is the desired behavior.
	msg := ev.(events.Message)
	// and then just call Notify, it's the same code anyway.
	request, err := NewRequest(msg.Type, msg.Actor.ID)
	if err != nil {
		return err
	}
	d.Notify(request)
	return nil
}

// pickObject selects and returns the next object to be processed. It returns
// the object event type and the object ID. If no objects remain, it will
// return noMoreObjects as the kind
//
// pickObjects picks objects in a specific order: Stack, Network, Secret,
// Config, and finally Service. Stacks must come first, because every other
// object type will depend on the latest stack, and Services must come last
// because they depend on the other object types. The middle 3 object types,
// Network, Secret, and Config, could be done in any order, but it's simpler
// to just assign them an order
func (d *dispatcher) pickObject() *interfaces.ReconcileResource {
	d.mu.Lock()
	defer d.mu.Unlock()
	for id, stack := range d.pendingStacks {
		// it should be safe to delete from a map we're iterating over.
		// especially considering we're not iterating any further.
		delete(d.pendingStacks, id)
		return stack
	}
	for id, nw := range d.pendingNetworks {
		delete(d.pendingNetworks, id)
		return nw
	}
	for id, secret := range d.pendingSecrets {
		delete(d.pendingSecrets, id)
		return secret
	}
	for id, config := range d.pendingConfigs {
		delete(d.pendingConfigs, id)
		return config
	}
	for id, service := range d.pendingServices {
		delete(d.pendingServices, id)
		return service
	}
	return nil
}
