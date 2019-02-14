package reconciler

// The reconciler package contains the parts of the stacks component that does
// the actual reconciliation of stacks. The reconciler's job is to make sure
// that the desired state as reflected in the Stack object is in turn reflected
// in the Specs spawned from that object.
//
// The reconciler package is tested with the Ginkgo BDD framework, because Drew
// (@dperny) wrote it and he likes Ginkgo.
