package reconciler

import (
	"github.com/docker/docker/errdefs"

	"github.com/docker/stacks/pkg/interfaces"
	"github.com/docker/stacks/pkg/reconciler/notifier"
)

// Reconciler is the interface implemented to do the actual work of computing
// and executing the changes required to bring the specs in line with
// those defined in the Stack.
type Reconciler interface {
	// Reconcile takes interfaces.ReconcileResource that may need to be
	// reconciled, and reconciles it. If it is a Stack, it may create new
	// objects and notify that changes have occurred. If the object is a
	// resource, like a service, belonging to a Stack, then it may be
	// updated or deleted to match the stack.
	//
	// Returns an error if the Resource cannot be reconciled, and nil if
	// successful.
	//
	Reconcile(resource *interfaces.ReconcileResource) error
}

// reconciler is the object that actually implements the Reconciler interface.
// reconciler is thread-safe, and is synchronous. This means tests for the
// reconciler can be written confined to one goroutine.
type reconciler struct {
	reconcileStackResource // nolint: unused
	notify                 notifier.ObjectChangeNotifier
	cli                    interfaces.BackendClient
	stackRequest           *reconcileStackRequest
}

// reconcileStackResource is a high-level interface to document the separation
// of dependencies which has been implemented.
type reconcileStackResource interface {
	getRequestedResource() *interfaces.ReconcileResource
	reconcile(stack interfaces.SnapshotStack) (interfaces.SnapshotStack, error)
}

// reconcileStackRequest is the top-level reconciliation datastructure for a Stack
type reconcileStackRequest struct {
	reconcileStackResource // nolint: unused
	requestedResource      *interfaces.ReconcileResource
	services               algorithmPlugin
	networks               algorithmPlugin
	secrets                algorithmPlugin
	configs                algorithmPlugin
}

// New creates a new Reconciler object, which uses the provided
// ObjectChangeNotifier and Client.
func New(notify notifier.ObjectChangeNotifier, cli interfaces.BackendClient) Reconciler {
	return newReconciler(notify, cli)
}

// newReconciler creates and returns a reconciler object. This returns the
// raw object, for use internally, instead of the interface as used externally.
func newReconciler(notify notifier.ObjectChangeNotifier, cli interfaces.BackendClient) *reconciler {
	r := &reconciler{
		notify: notify,
		cli:    cli,
	}
	return r
}

// The dispatcher algorithm reschedules the reconciliation request if
// there is an error.  Given the implementation below, the Reconcile
// function is NOT idempotent.  The dispatcher will decide
// what to do with reconciliation requests that repeatedly fail.
//
// PLEASE NOTE: The calls to this function are caused only by the
// dispatcher because of an external-to-reconciler event.
func (r *reconciler) Reconcile(request *interfaces.ReconcileResource) error {

	r.stackRequest = nil

	serviceInit := newInitializationSupportService(r.cli)
	secretInit := newInitializationSupportSecret(r.cli)
	networkInit := newInitializationSupportNetwork(r.cli)
	configInit := newInitializationSupportConfig(r.cli)

	var algorithmInit initializationSupport

	if request.StackID == "" {
		switch request.Kind {

		case interfaces.ReconcileStack:
			request.StackID = request.ID
			break
		case interfaces.ReconcileService:
			algorithmInit = &serviceInit
			break
		case interfaces.ReconcileSecret:
			algorithmInit = &secretInit
			break
		case interfaces.ReconcileNetwork:
			algorithmInit = &networkInit
			break
		case interfaces.ReconcileConfig:
			algorithmInit = &configInit
			break
		}
		if algorithmInit != nil {
			resource, err := algorithmInit.getActiveResource(*request)
			if errdefs.IsNotFound(err) {
				// If the resource isn't found,
				// that means some other mutator is active and
				// another reconciler approach is required
				//
				// FIXME: Add reconciler statistic
				return nil
			} else if err != nil {
				return err
			}

			if resource.getStackID() == "" {
				// If the resource stack label is not found,
				// that means some other mutator is active and
				// another reconciler approach is required
				//
				// FIXME: Add reconciler statistic
				return nil
			}
			request.StackID = resource.getStackID()
		}
	}

	snapshot, err := r.cli.GetSnapshotStack(request.StackID)
	if err != nil {
		if errdefs.IsNotFound(err) {
			// If the snapshot is gone, there is no bookkeeping
			// work to perform. The algorithm is intended to
			// capture orphaned labels long before this moment
			return nil
		}
		return err
	}

	// Currently, even if a reconcile request Kind was for a non-Stack, all
	// of the types.Stack resources are processed and the original
	// reconcile request is passed to all types.Stack resources.  This
	// permits some more sophisticated dependency management if needed.

	r.stackRequest = &reconcileStackRequest{
		requestedResource: request,
		services:          serviceInit.createPlugin(snapshot, request),
		secrets:           secretInit.createPlugin(snapshot, request),
		networks:          networkInit.createPlugin(snapshot, request),
		configs:           configInit.createPlugin(snapshot, request),
	}

	_, err = r.reconcile(snapshot)

	return err
}

func (r *reconciler) getRequestedResource() *interfaces.ReconcileResource {
	return r.stackRequest.getRequestedResource()
}

func (r *reconciler) reconcile(stack interfaces.SnapshotStack) (interfaces.SnapshotStack, error) {
	return r.stackRequest.reconcile(stack)
}

func (r reconcileStackRequest) getRequestedResource() *interfaces.ReconcileResource {
	return r.requestedResource
}

func (r reconcileStackRequest) reconcile(stack interfaces.SnapshotStack) (interfaces.SnapshotStack, error) {

	// FIXME: GIVEN HOW THE NEW PIECES FIT TOGETHER, THIS SITUATION
	//        NEEDS A DISCUSSION, the stack store API can trigger
	//        the event itself
	//
	//	if ... {
	//		return ...cli.DeleteStack(stackID)
	//	}

	var err error
	stack, err = r.secrets.reconcile(stack)
	if err != nil {
		return stack, err
	}
	stack, err = r.configs.reconcile(stack)
	if err != nil {
		return stack, err
	}
	stack, err = r.networks.reconcile(stack)
	if err != nil {
		return stack, err
	}
	stack, err = r.services.reconcile(stack)
	if err != nil {
		return stack, err
	}
	return stack, nil
}
