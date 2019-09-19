package reconciler

// nolint: golint
import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/docker/stacks/pkg/fakes"
	"github.com/docker/stacks/pkg/interfaces"
	"github.com/docker/stacks/pkg/types"
)

// AlgorithmPluginInputs is shared state for shared Ginko tests
type AlgorithmPluginInputs struct {
	algorithmInit                    initializationSupport
	resourcePlugin                   algorithmPlugin
	err1, err2                       error
	search1, search2                 interfaces.ReconcileResource
	activeResource1, activeResource2 activeResource
	stackID                          string
	request                          *interfaces.ReconcileResource
	stack                            *types.Stack
	specName                         string
	resource                         *interfaces.ReconcileResource
	activeResource                   activeResource
	cli                              *fakes.FakeReconcilerClient
}

// SharedFailedResponseBehavior placeholder
func SharedFailedResponseBehavior(inputs *AlgorithmPluginInputs) {
	BeforeEach(func() {
		inputs.activeResource1, inputs.err1 =
			inputs.algorithmInit.getActiveResource(inputs.search1)
		inputs.activeResource2, inputs.err2 =
			inputs.algorithmInit.getActiveResource(inputs.search2)
	})
	Context("Testing 5 6 7", func() {
		It("the UN-LABELED resource should not fail", func() {
			Expect(inputs.err1).ToNot(HaveOccurred())
			Expect(inputs.activeResource1.getSnapshot().Name).To(Equal(inputs.search1.Name))
			Expect("").To(Equal(inputs.activeResource1.getStackID()))
		})
	})
	Context("Testing 1 2 4", func() {
		It("the LABELED resource should not fail", func() {
			Expect(inputs.err2).ToNot(HaveOccurred())
			Expect(inputs.activeResource2.getSnapshot().Name).To(Equal(inputs.search2.Name))
			Expect("").ToNot(Equal(inputs.activeResource2.getStackID()))
			Expect(inputs.stackID).To(Equal(inputs.activeResource2.getStackID()))
		})
	})
}

// FreshPluginAssertions placeholder
func FreshPluginAssertions(input *AlgorithmPluginInputs) {

	It("Preconditions Met", func() {
		Expect(input.specName).ToNot(Equal(""))
	})

	It("No Goals set", func() {
		noGoals := input.resourcePlugin.getGoalResources()
		Expect(noGoals).To(BeEmpty())
	})

	It("No particular goal set", func() {
		noGoal := input.resourcePlugin.getGoalResource(input.specName)
		Expect(noGoal).To(BeNil())
	})

	It("Specified Resources Match", func() {
		specNames := input.resourcePlugin.getSpecifiedResourceNames()
		Expect(specNames).To(HaveLen(1))
	})

	It("No ACTIVE resources findable", func() {
		noActiveResources, err := input.resourcePlugin.getActiveResources()
		Expect(err).ToNot(HaveOccurred())
		Expect(noActiveResources).To(BeEmpty())
	})
}

// CreatedGoalAssertions placeholder
func CreatedGoalAssertions(input *AlgorithmPluginInputs) {

	// NOW USE resourcePlugin to alter resourcePlugin ONLY
	// STILL NO ACTIVE RESOURCES
	BeforeEach(func() {
		input.resource =
			input.resourcePlugin.addCreateResourceGoal(input.specName)
	})

	It("Added goal only added not created", func() {
		Expect(input.resource.Name).To(Equal(input.specName))
		Expect(input.resource.ID).To(Equal(""))
	})

	It("Goals set", func() {
		goals := input.resourcePlugin.getGoalResources()
		Expect(goals).ToNot(BeEmpty())
	})

	It("particular goal set", func() {
		aGoal := input.resourcePlugin.getGoalResource(input.specName)
		Expect(input.resource == aGoal).To(BeTrue())
	})

	It("Specified Resources Match", func() {
		specNames := input.resourcePlugin.getSpecifiedResourceNames()
		Expect(specNames).To(HaveLen(1))
	})

	It("No ACTIVE resources findable", func() {
		noActiveResources, err := input.resourcePlugin.getActiveResources()
		Expect(err).ToNot(HaveOccurred())
		Expect(noActiveResources).To(BeEmpty())
	})
}

// ManuallyReconcileGoal placeholder
func ManuallyReconcileGoal(input *AlgorithmPluginInputs) {
	var (
		createErr error
	)

	BeforeEach(func() {
		createErr = input.resourcePlugin.createResource(input.resource)
	})

	It("resource creation updates resource pointer", func() {
		Expect(createErr).ToNot(HaveOccurred())
		Expect(input.resource.ID).ToNot(Equal(""))
	})

	It("ACTIVE resources findable", func() {
		activeResources, err := input.resourcePlugin.getActiveResources()
		Expect(err).ToNot(HaveOccurred())
		Expect(activeResources).ToNot(BeEmpty())
	})

	It("ACTIVE resource findable", func() {
		var err error
		input.activeResource, err = input.resourcePlugin.getActiveResource(*input.resource)
		Expect(err).ToNot(HaveOccurred())
		Expect(input.activeResource.getSnapshot().ID).To(Equal(input.resource.ID))
	})

	It("ACTIVE resource matches configuration", func() {
		same := input.resourcePlugin.hasSameConfiguration(*input.resource, input.activeResource)
		Expect(same).To(BeTrue())
	})
}

// ManuallyStoreGoal placeholder
func ManuallyStoreGoal(input *AlgorithmPluginInputs) {
	var (
		current, snapshot interfaces.SnapshotStack
		err               error
	)
	BeforeEach(func() {
		var errSnapshot error
		snapshot, errSnapshot = input.cli.GetSnapshotStack(input.stackID)
		Expect(errSnapshot).ToNot(HaveOccurred())
		current, err = input.resourcePlugin.storeGoals(snapshot)
	})
	It("And snapshot resources update", func() {
		Expect(err).ToNot(HaveOccurred())
		Expect(input.resourcePlugin.getSnapshotResourceNames(snapshot)).To(HaveLen(0))
		Expect(snapshot.Meta.Version.Index != current.Meta.Version.Index).To(BeTrue())
		Expect(input.resourcePlugin.getSnapshotResourceNames(current)).To(HaveLen(1))
	})
}

// ManuallyRemoveGoal placeholder
func ManuallyRemoveGoal(input *AlgorithmPluginInputs) {
	var (
		resource *interfaces.ReconcileResource
		activeResource activeResource
		err error
	)
	BeforeEach(func() {
		activeResource, err = input.resourcePlugin.getActiveResource(*input.resource)
		Expect(err).ToNot(HaveOccurred())
		resource = input.resourcePlugin.addRemoveResourceGoal(activeResource)		
	})
	It("And snapshot resources update", func() {
		Expect(resource).ToNot(BeNil())
		Expect(resource.SnapshotResource).ToNot(BeNil())
		Expect(activeResource).ToNot(BeNil())
		Expect(activeResource.getSnapshot()).ToNot(BeNil())
		Expect(resource.SnapshotResource).To(Equal(activeResource.getSnapshot()))
		Expect(input.resourcePlugin.getGoalResources()).To(ContainElement(resource))
	})
}

// ManuallyReconcileRemoveGoal placeholder
func ManuallyReconcileRemoveGoal(input *AlgorithmPluginInputs) {
	var (
		err error
	)
	BeforeEach(func() {
		err = input.resourcePlugin.deleteResource(input.resource)		
	})
	It("And snapshot resources update", func() {
		Expect(err).ToNot(HaveOccurred())
	})
}

// ManuallyReconcileUpdateGoal placeholder
func ManuallyReconcileUpdateGoal(input *AlgorithmPluginInputs) {
	var (
		err error
	)
	BeforeEach(func() {
		act, _ := input.resourcePlugin.getActiveResource(*input.resource)
		input.resource.Meta = act.getSnapshot().Meta
		err = input.resourcePlugin.updateResource(*input.resource)		
	})
	It("And snapshot resources update", func() {
		Expect(err).ToNot(HaveOccurred())
	})
}
