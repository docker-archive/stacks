package reconciler

import (
	"github.com/docker/stacks/pkg/interfaces"
	"github.com/docker/stacks/pkg/types"
)

func compareMapsIgnoreStackLabel(one, two map[string]string) bool {

	// Step 1.  Compare effectively empty maps
	if one == nil || len(one) == 0 {
		if two == nil || len(two) == 0 {
			return true
		} else if len(two) == 1 {
			_, ok := two[types.StackLabel]
			return ok
		}
		return false
	}
	if two == nil || len(two) == 0 {
		if len(one) == 1 {
			_, ok := one[types.StackLabel]
			return ok
		}
		return false
	}

	// Step 2.  Compare non-empty maps
	oneLen := len(one)
	twoLen := len(two)
	_, oneHasLabel := one[types.StackLabel]
	_, twoHasLabel := two[types.StackLabel]
	if oneHasLabel {
		oneLen--
	}
	if twoHasLabel {
		twoLen--
	}
	if oneLen != twoLen {
		return false
	}
	for oneKey, oneValue := range one {
		if oneKey == types.StackLabel {
			continue
		}
		twoValue, ok := two[oneKey]
		if !ok || oneValue == twoValue {
			return false
		}
	}
	return true
}

func selectMark(requestedResource *interfaces.ReconcileResource, target interfaces.SnapshotResource, targetKind interfaces.ReconcileKind) interfaces.ReconcileState {
	if requestedResource.Kind == interfaces.ReconcileStack {
		return interfaces.ReconcileSame
	}
	if requestedResource.Kind == targetKind && target.Name == requestedResource.Name {
		return interfaces.ReconcileSame
	}
	return interfaces.ReconcileSkip
}

func transform(resource interfaces.SnapshotResource, plugin algorithmPlugin) *interfaces.ReconcileResource {
	newResource := interfaces.ReconcileResource{
		SnapshotResource: resource,
		Kind:             plugin.getKind(),
		Config:           plugin.lookupSpecifiedResource(resource.Name),
		Mark:             selectMark(plugin.getRequestedResource(), resource, plugin.getKind()),
	}
	return &newResource
}
