package reconciler

import (
	"github.com/docker/docker/api/types/filters"

	"github.com/docker/stacks/pkg/interfaces"
)

/**
 *  Reconciler algorithm.
 *
 *  See https://en.wikipedia.org/wiki/Balking_pattern
 *
 *  The resources collected under a types.Stack are NEITHER intended NOR
 *  capable to be exclusively locked by the reconciler for its operation.
 *
 *  However, if those resources were exclusively lockable, the reconciler
 *  would still have to implement a complicated query under a read-lock
 *  before attempting a write-lock.
 *
 *  The reconciler algorithm instead uses the so-called "Balking Pattern"
 *  to emulate the read-lock / write-lock discipline in order to make changes
 *  to resources (service, config, network, secret, stack).  If nothing has
 *  caused a "Balk" to occur, the decision to make external changes is
 *  justified.
 *
 *  The query implementing the Balk collects information:
 *  1. Which resources require external changes
 *  2. Evidence that another MUTATOR (reconciler, an independent
 *     resource alteration) has been active during the query
 *
 *  Therefore the query is responsible for
 *  1. Limiting the number of proposed external changes/interactions (which
 *     is consistent with the intentions of a read/write lock pattern).
 *  2. Requesting a query retry because a MUTATOR has altered the consistency
 *     of the query (which is what a read-lock would prevent).
 *
 *  Using the query, the Balk itself is implemented thusly:
 *  A. The GOAL state of the reconciler's actions are durably stored
 *     simultaneously with types.Stack BEFORE reconciliation actions occur
 *  B. If the computed GOAL state conflicts with the previous GOAL state,
 *     then the changes to the underlying resources are implied
 *  C. If the previous GOAL state has changed during computing a new GOAL
 *     state, then i) another reconciliation algorithm is active, the
 *     ii) the computed GOAL state is stale
 *
 *  GOAL State:  The Name, ID and swarm.Meta of the resources specified in
 *               types.Stack.  The ID may not be available for individual
 *               resources in certain GOAL States.  swarm.Meta is for
 *               bookkeeping
 *
 *  Implementation Note:  At the time of writing, the GOAL state is not
 *  relying on swarm.Meta to compare the specification versions in order to
 *  detect changes. That would require a greater contract on the UPDATE API
 *  calls to return the the new version and/or object.  Tools like
 *  reflect.DeepEquals to are needed to compare specifications stored in the
 *  stack and in the resource
 */

// activeResource is an ephemeral wrapper meant to carry the GOAL identifiers
// about a resource and the queried types.Stack resource (service, config,
// network, secret).  This wrapper separates the algorithm from
// the representation.  There are four implementations of this interface.
type activeResource interface {
	getSnapshot() interfaces.SnapshotResource
	getStackID() string
}

// algorithmPlugin hides the Resource specific minutiae from the algorithm's
// progress.  This interface represents that
// 1. the algorithm needs to compare the current spec, the active resource
//    and maintain the goal set.
// 2. add and store goals
// 3. change resources
// There are four implementations of this interface.
type initializationSupport interface {
	getActiveResource(interfaces.ReconcileResource) (activeResource, error)
	// for testing
	getSnapshotResourceNames(interfaces.SnapshotStack) []string
	getKind() interfaces.ReconcileKind
	createPlugin(interfaces.SnapshotStack, *interfaces.ReconcileResource) algorithmPlugin
}
type algorithmPlugin interface {
	initializationSupport
	reconcileStackResource

	getSpecifiedResourceNames() []string
	lookupSpecifiedResource(name string) interface{}

	getActiveResources() ([]activeResource, error)

	getGoalResources() []*interfaces.ReconcileResource
	getGoalResource(name string) *interfaces.ReconcileResource

	// FIXME: Clarify the precondition, resource.ID != ""
	hasSameConfiguration(resource interfaces.ReconcileResource, actual activeResource) bool

	addCreateResourceGoal(specName string) *interfaces.ReconcileResource
	addRemoveResourceGoal(resource activeResource) *interfaces.ReconcileResource
	storeGoals(interfaces.SnapshotStack) (interfaces.SnapshotStack, error)

	// createResource creates the Docker resource AND updates resource.ID
	createResource(*interfaces.ReconcileResource) error

	// deleteResource deletes the Docker resource AND erases resource.ID
	deleteResource(resource *interfaces.ReconcileResource) error

	// updateResource updates the Docker resource ONLY
	updateResource(interfaces.ReconcileResource) error
}

// stackLabelFilter constructs a filter.Args which filters for stacks based on
// the stack label being equal to the stack ID.
func stackLabelFilter(stackID string) filters.Args {
	return filters.NewArgs(interfaces.StackLabelArg(stackID))
}

// reconcileResource implements the reconciliation pattern using an
// individual algorithmPlugin interface
// nolint: gocyclo
func reconcileResource(current interfaces.SnapshotStack, plugin algorithmPlugin) (interfaces.SnapshotStack, error) {

	// QUERY (see file comment)
	//
	// 1. If all the resources in a Stack can be marked as requiring NO
	// Create, Update or Delete calls to Docker APIs, aka SAME, then
	// fast-path reconciliation is done AND no alterations are needed.
	//
	// 2. If any resource requires a Create, Update or Delete Docker API
	// call, then a write-lock is requested to store the current state of
	// the reconciler's computed GOAL state on Docker API's.
	//
	// 3. If the GOAL state write fails, then a MUTATOR is detected
	// This function requests a policy based retry of reconciliation
	// on account of a stale GOAL xor API call anomaly, which constitute
	// a MUTATOR
	//
	// 4. If reconciler's GOAL state write succeeds, then the
	// Balking algorithm has provided an exclusionary
	// window of opportunity for a resource Create, Update or Delete
	// Docker API call as the only reconciler calling.
	// Failures imply a policy based retry of reconciliation

	// MARK PHASE 1 - Initially mark as DELETE all previous resources
	//                unless requested to SKIP a resource reconciliation
	//
	for _, resource := range plugin.getGoalResources() {
		if resource.Mark != interfaces.ReconcileSkip {
			resource.Mark = interfaces.ReconcileDelete
		}
	}

	//
	// MARK PHASE 2 - Mark COMPARE any DELETE Resources when
	//                a current specification is found
	//              - Mark new specifications for CREATE
	//
	for _, specName := range plugin.getSpecifiedResourceNames() {
		resource := plugin.getGoalResource(specName)
		if resource != nil {
			if resource.Mark == interfaces.ReconcileDelete {
				resource.Mark = interfaces.ReconcileCompare
			}
		} else {
			added := plugin.addCreateResourceGoal(specName)
			added.Mark = interfaces.ReconcileCreate

		}
	}

	// At this point, the goal resources are marked one of the following:
	// SKIP, DELETE, COMPARE, CREATE

	//
	// MARK PHASE 3 - Query for all active Resources labelled as belonging
	//                to the Stack
	//		- Commonly, active Resources will match COMPARE marks
	//              - Update stale Resource ID's
	//              - Perhaps, Active Resource matches an above CREATE
	//              - DELETE Active Resource without specification and
	//                not previously recorded
	//              - Compare active specification to Stack specification
	//                and mark SAME xor UPDATE
	//
	activeResources, err := plugin.getActiveResources()
	if err != nil {
		return current, err
	}

	for _, activeResourceWrapper := range activeResources {
		activeResource := activeResourceWrapper.getSnapshot()
		resource := plugin.getGoalResource(activeResource.Name)
		if resource != nil {

			if resource.Mark == interfaces.ReconcileCreate {
				// MATCHING CREATE -  Implies another MUTATOR
				// created the to-be-created resource.
				//
				// The implementation in PHASE 4 will clean
				// this up
				resource.Mark = interfaces.ReconcileCompare
				resource.ID = activeResource.ID
				resource.Meta = activeResource.Meta

			} else if resource.Mark == interfaces.ReconcileDelete {
				// Name matches previous goal, spec missing
				resource.ID = activeResource.ID
				resource.Meta = activeResource.Meta

			} else if resource.Mark == interfaces.ReconcileCompare {
				if plugin.hasSameConfiguration(*resource, activeResourceWrapper) {
					resource.Mark = interfaces.ReconcileSame
				} else {
					resource.Mark = interfaces.ReconcileUpdate
				}
				resource.ID = activeResource.ID
				resource.Meta = activeResource.Meta
			}

		} else {

			// MISMATCHING DELETE -  Implies another MUTATOR
			// Name does not match previous goal, spec missing
			//
			// The resource is an orphan to be removed
			// or it is improperly associated with the stack
			added := plugin.addRemoveResourceGoal(activeResourceWrapper)
			added.Mark = interfaces.ReconcileDelete

		}
	}

	// At this point, the goal resources are marked one of the following:
	// SKIP, DELETE, COMPARE, CREATE, UPDATE, SAME
	//
	// 1. Anything still marked DELETE from Phase ONE has neither an active
	//    resource nor spec owning why the previous GOAL record exists.
	//    The GOAL record is to be DELETE'd AND the Resource, if still
	//    existing, is to be DELETE'd
	//
	// 2. Anything still marked COMPARE has not been found via the query in
	//    Phase THREE and needs i) re-CREATE AND ii) if the resource.ID
	//    still exists the resource will be deleted.
	//
	// 3. Otherwise the marks from Phase THREE represent expected
	//    courses of action.
	//
	// 4. Therefore if all Resources are marked SAME, then
	// - no deletions are pending
	// - no creations are pending
	// - no updates are pending
	// Then this is the fast, common and expected path.
	//
	// PHASE FOUR - Mark COMPARE Resources for re-CREATE
	//            - Determine if all Services are SAME
	//

	witnessedAllSame := true
	for _, resource := range plugin.getGoalResources() {
		if resource.Mark == interfaces.ReconcileCompare {
			/*
			 * PHASE 3 guarantees the following commented code to
			 * be deadcode.  It is included here for documentation
			 * purposes.
			 *
			 * for _, activeResource := range activeResources {
			 *	if activeResource.getSnapshot().Name == resource.Name {
			 *		return current, errdefs.Forbidden("Resource inexplicably found")
			 *	}
			 * }
			 */
			// FIXME: If the previous ID of this resource still
			// exists then perhaps this scenario is an update.
			// It is possible there is a MUTATOR at work and
			// security is to delete and re-create.
			if resource.ID != "" {
				// ignore error
				_ = plugin.deleteResource(resource)
			}
			resource.Mark = interfaces.ReconcileCreate
			resource.ID = ""
			witnessedAllSame = false
		} else if resource.Mark != interfaces.ReconcileSame {
			witnessedAllSame = false
		}
	}

	if witnessedAllSame {
		return current, nil
	}
	// GOAL CONTRACT
	//
	// This will store currently known Resources related to this Stack
	// in order to prepare the next iteration BEFORE the actual Resource
	// CREATE, UPDATE and DELETE options are attempted.
	//
	//

	var storeError error
	current, storeError = plugin.storeGoals(current)
	if storeError != nil {
		return current, storeError
	}

	// At this point, the GOAL resources are marked one of the following:
	// SKIP, DELETE, CREATE, UPDATE, SAME

	var mutationError error
	for _, resource := range plugin.getGoalResources() {
		if resource.Mark == interfaces.ReconcileSkip {
			continue
		}
		if resource.Mark == interfaces.ReconcileSame {
			continue
		}
		if resource.Mark == interfaces.ReconcileCreate {
			// FIXME: One potential error condition is a name
			// collision with an existing resource not found in
			// this stack
			mutationError = plugin.createResource(resource)
		} else if resource.Mark == interfaces.ReconcileDelete {
			mutationError = plugin.deleteResource(resource)
		} else if resource.Mark == interfaces.ReconcileUpdate {
			mutationError = plugin.updateResource(*resource)
		}
		if mutationError == nil {
			// Depending on the optimisms of the implemented balk
			// this storeGoals call can be moved out of the loop
			// but it weakens the protocol
			current, mutationError = plugin.storeGoals(current)
		}
		if mutationError != nil {
			return current, mutationError
		}
	}

	return current, nil
}
