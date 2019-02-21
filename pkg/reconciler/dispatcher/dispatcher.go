package dispatcher

import (
	"sync"

	"github.com/docker/docker/api/types/events"

	"github.com/docker/stacks/pkg/interfaces"
	"github.com/docker/stacks/pkg/reconciler/notifier"
	"github.com/docker/stacks/pkg/reconciler/reconciler"
)

const (
	noMoreObjects = "none left"
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

	// currently, the reconciler package only works with Stacks. The dispatcher
	// will be updated to handle more object types as the Reconciler implements
	// functionality for them.

	// pendingStacks (and the similar pending maps) are sets of object IDs. at
	// first glance, we might want to put all objects into a map[string]string,
	// where the key is the ID and the value is the kind. however, we have to
	// reconcile objects in order: stacks, then networks, configs, and secrets,
	// and finally services.
	pendingStacks map[string]struct{}
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
		r:             r,
		pendingStacks: map[string]struct{}{},
	}
	register.Register(m)
	return m
}

// Notify tells the dispatcher to call the reconciler with this object at some
// point in the future
func (d *dispatcher) Notify(kind, id string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	// TODO(dperny) implement
	switch kind {
	case interfaces.StackEventType:
		d.pendingStacks[id] = struct{}{}
	case events.ServiceEventType:
		// TODO(dperny): we don't handle service events, so we don't yet
		// implement this case.
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
		d.resolveMessage(ev)

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
				d.resolveMessage(ev)
			default:
				// when the channel is no longer ready, process an event
				kind, id := d.pickObject()
				if kind == noMoreObjects {
					// if there are no more objects in the queue, go back to
					// waiting for an event
					break readingEvents
				}
				// next state: reconcile the object. if it fails, add it back
				// to the set of objects.
				if err := d.r.Reconcile(kind, id); err != nil {
					// TODO(dperny): if a given object always fails, we'll stay
					// in this state forever, looping again and again.
					d.Notify(kind, id)
				}
			}
		}
	}
}

// resolveMessage is a method that figures out what kind of event this is and
// puts it into the correct map
func (d *dispatcher) resolveMessage(ev interface{}) {
	// naked type cast. If this isn't events.Message, then the program will
	// panic. This is the desired behavior.
	msg := ev.(events.Message)
	// and then just call Notify, it's the same code anyway.
	d.Notify(msg.Type, msg.Actor.ID)
}

// pickObject selects and returns the next object to be processed. It returns
// the object event type and the object ID. If no objects remain, it will
// return noMoreObjects as the kind
func (d *dispatcher) pickObject() (string, string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	// first, do we have any Stacks ready?
	for stack := range d.pendingStacks {
		// it should be safe to delete from a map we're iterating over.
		// especially considering we're not iterating any further.
		delete(d.pendingStacks, stack)
		return interfaces.StackEventType, stack
	}
	return noMoreObjects, ""
}
