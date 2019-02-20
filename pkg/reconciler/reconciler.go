package reconciler

import (
	"fmt"

	"github.com/docker/stacks/pkg/interfaces"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/errdefs"
)

const (
	// StackLabel defines the label indicating that a resource belongs to a
	// particular stack.
	StackLabel = "com.docker.stacks.stack_id"

	// StackEventType defines the string indicating that an event is a stack
	// event
	StackEventType = "stack"
)

// Client is the subset of interfaces.BackendClient methods needed to
// implement the Reconciler.
type Client interface {
	// stack methods
	GetSwarmStack(string) (interfaces.SwarmStack, error)

	// service methods
	GetServices(dockerTypes.ServiceListOptions) ([]swarm.Service, error)
	GetService(string, bool) (swarm.Service, error)
	CreateService(swarm.ServiceSpec, string, bool) (*dockerTypes.ServiceCreateResponse, error)

	// TODO(dperny): there's a lot more where this came from, but these are the
	// parts we need to make this part go
}

// Reconciler is the interface implemented to do the actual work of computing
// and executing the changes required to bring the cluster's specs in line with
// those defined in the Stack.
type Reconciler interface {
	// Reconcile takes the Kind and ID of an object that may need to be
	// reconciled, and reconciles it. If it is a Stack, it may create new
	// objects and notify that changes have occurred. If the object is a
	// resource, like a service, belonging to a Stack, then it may be updated
	// or deleted to match the stack.
	//
	// Returns an error if the Resource cannot be reconciled, and nil if
	// successful.
	//
	// TODO(dperny): we may actually want to pass a whole
	// (github.com/docker/docker/types/events.Message) object to this, instead
	// of an ID and Kind. That would allow us to optimize our decision on
	// whether or not there is any reconciliation that needs to be done. I've
	// punted on doing so for now for simplicity's sake. We'll optimize later.
	Reconcile(kind, id string) error

	// Deletetakes the ID of an object that has been deleted. If the object is
	// a stack, the reconciler then calls the ObjectChangeNotifier with all of
	// the resources belonging to that stack, which will cause them to be
	// deleted in turn. Otherwise, if the object was deleted in error, it may
	// be recreated.
	Delete(kind, id string) error
}

// ObjectChangeNotifier is an interface defining an object that can be called
// back to if the Reconciler decides that it needs to take another pass at some
// object. The ObjectChangeNotifier may seem a bit excessive, but it provides
// the key functionality of decoupling the synchronous part of the Reconciler
// from the asynchronous part of the component that calls into it. Without it,
// the Reconciler might have both synchronous and asynchronous components in
// the same object (a pattern common in Swarmkit), which would make testing
// much more difficult.
type ObjectChangeNotifier interface {
	// Notify indicates the kind and ID of an object that should be reconciled
	Notify(kind, id string)
}

// reconciler is the object that actually implements the Reconciler interface.
// reconciler is thread-safe, and is synchronous. This means tests for the
// reconciler can be written confined to one goroutine.
type reconciler struct {
	notifier ObjectChangeNotifier
	cli      Client
}

// NewReconciler creates a new Reconciler object, which uses the provided
// ObjectChangeNotifier and Client.
func NewReconciler(notifier ObjectChangeNotifier, cli Client) Reconciler {
	return newReconciler(notifier, cli)
}

// newReconciler creates and returns a reconciler object. This returns the
// raw object, for use internally, instead of the interface as used externally.
func newReconciler(notifier ObjectChangeNotifier, cli Client) *reconciler {
	r := &reconciler{
		notifier: notifier,
		cli:      cli,
	}
	return r
}

func (r *reconciler) Reconcile(kind, id string) error {
	switch kind {
	case StackEventType:
		return r.reconcileStack(id)
	default:
		// TODO(dperny): what if it's none of these?
		return nil
	}
}

// reconcileStack implements the ReconcileStack method of the Reconciler
// interface
func (r *reconciler) reconcileStack(id string) error {
	stack, err := r.cli.GetSwarmStack(id)
	switch {
	case errdefs.IsNotFound(err):
		return nil
	case err != nil:
		return err
	}

	for _, spec := range stack.Spec.Services {
		// try getting the service to see if it already exists
		service, err := r.cli.GetService(spec.Annotations.Name, false)
		// if it doesn't exist create it now
		if errdefs.IsNotFound(err) {
			// TODO(dperny): second 2 arguments?
			// TODO(dperny): we don't cache service data right now, but we
			// might want to do so later
			_, err := r.cli.CreateService(spec, "", false)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			// if the service already exists, it should be reconciled after
			// this, so notify
			r.notifier.Notify("service", service.ID)
		}
	}

	return nil
}

// Delete takes the kind and ID of an object that has been deleted and
// reconciles it
func (r *reconciler) Delete(kind, id string) error {
	switch kind {
	case StackEventType:
		return r.deleteStack(id)
	default:
		// TODO(dperny): implement for other kinds
		return nil
	}
}

func (r *reconciler) deleteStack(id string) error {
	// it doesn't matter if the stack is actually deleted or not, so we don't
	// have to get it from the backend. If it isn't deleted, the services will
	// not be deleted when we reconcile them in a bit.
	//
	// We do have to get all services labeled for this stack
	services, err := r.cli.GetServices(dockerTypes.ServiceListOptions{Filters: stackLabelFilter(id)})
	if err != nil {
		return err
	}
	for _, service := range services {
		r.notifier.Notify("service", service.ID)
	}
	return nil
}

// stackLabelFilter constructs a filter.Args which filters for stacks based on
// the stack label being equal to the stack ID.
func stackLabelFilter(stackID string) filters.Args {
	return filters.NewArgs(
		filters.Arg("label", fmt.Sprintf("%s=%s", StackLabel, stackID)),
	)
}
