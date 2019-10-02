package reconciler

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	//	"github.com/docker/docker/api/types/swarm"

	"github.com/docker/stacks/pkg/fakes"
	"github.com/docker/stacks/pkg/interfaces"
	"github.com/docker/stacks/pkg/types"
)

var _ = Describe("Algorithm Plugin for Service - Stack Request", func() {
	var (
		serviceInit    initializationService
		stack          types.Stack
		input          AlgorithmPluginInputs
		err1, err2     error
		snapshot       interfaces.SnapshotStack
		serviceSupport algorithmService
	)
	BeforeEach(func() {
		input.cli = fakes.NewFakeReconcilerClient()
		serviceInit = newInitializationSupportService(input.cli)
		stack = fakes.GetTestStack("stack12")
		input.stack = &stack
		input.algorithmInit = serviceInit
		input.stackID, err1 = input.cli.AddStack(input.stack.Spec)
		snapshot, err2 = input.cli.FakeStackStore.GetSnapshotStack(input.stackID)
		input.specName = input.stack.Spec.Services[0].Annotations.Name
	})
	It("Initializations Succeed", func() {
		Expect(err1).ToNot(HaveOccurred())
		Expect(err2).ToNot(HaveOccurred())
	})
	BeforeEach(func() {
		input.request = &interfaces.ReconcileResource{
			SnapshotResource: interfaces.SnapshotResource{
				ID: input.stackID,
			},
			Kind: interfaces.ReconcileStack,
		}

		serviceSupport := newAlgorithmPluginService(serviceInit, snapshot, input.request)
		input.resourcePlugin = serviceSupport
	})
	Context("With a fresh plugin", func() {
		It("Coverage missing lookup", func() {
			noSpecName := serviceSupport.lookupServiceSpec("missing")
			Expect(noSpecName).To(BeNil())
		})
		FreshPluginAssertions(&input)
	})
	Context("Add a creation goal", func() {
		CreatedGoalAssertions(&input)

		Context("Manually reconcile goal", func() {
			ManuallyReconcileGoal(&input)

			Context("Manually store goal", func() {
				ManuallyStoreGoal(&input)
			})
			Context("Manually update goal", func() {
				ManuallyReconcileUpdateGoal(&input)
			})
			Context("Manually remove goal", func() {
				ManuallyRemoveGoal(&input)
				Context("Manually remove goal", func() {
					ManuallyReconcileRemoveGoal(&input)
				})
			})
		})
	})
	Context("Provide an update scenario", func() {
		BeforeEach(func() {
			CreatedGoalAssertions(&input)
			ManuallyReconcileGoal(&input)
			ManuallyStoreGoal(&input)
			/* for another tests
			// Manual alteration of underlying service
			input.stack.Spec.Services[0].UpdateConfig =
				&swarm.UpdateConfig{}
			service, _ := input.cli.GetService(input.activeResource.getSnapshot().ID,
						interfaces.DefaultGetServiceArg2)
			input.cli.UpdateService(service.ID,
						service.Meta.Version.Index,
						input.stack.Spec.Services[0],
						interfaces.DefaultUpdateServiceArg4,
						interfaces.DefaultUpdateServiceArg5)
			snapshot, err2 = input.cli.FakeStackStore.GetSnapshotStack(input.stackID)
			serviceSupport := newAlgorithmPluginService(serviceInit, snapshot, input.request)
			input.resourcePlugin = serviceSupport
			*/

		})

	})
})

var _ = Describe("Algorithm Plugin for Config - Stack Request", func() {
	var (
		configInit    initializationConfig
		stack         types.Stack
		input         AlgorithmPluginInputs
		err1, err2    error
		snapshot      interfaces.SnapshotStack
		configSupport algorithmConfig
	)
	BeforeEach(func() {
		input.cli = fakes.NewFakeReconcilerClient()
		configInit = newInitializationSupportConfig(input.cli)
		stack = fakes.GetTestStack("stack12")
		input.stack = &stack
		input.algorithmInit = configInit
		input.stackID, err1 = input.cli.AddStack(input.stack.Spec)
		snapshot, err2 = input.cli.FakeStackStore.GetSnapshotStack(input.stackID)
		input.specName = input.stack.Spec.Configs[0].Annotations.Name
	})
	It("Initializations Succeed", func() {
		Expect(err1).ToNot(HaveOccurred())
		Expect(err2).ToNot(HaveOccurred())
	})
	BeforeEach(func() {
		input.request = &interfaces.ReconcileResource{
			SnapshotResource: interfaces.SnapshotResource{
				ID: input.stackID,
			},
			Kind: interfaces.ReconcileStack,
		}

		configSupport := newAlgorithmPluginConfig(configInit, snapshot, input.request)
		input.resourcePlugin = configSupport
	})
	Context("With a fresh plugin", func() {
		It("Coverage missing lookup", func() {
			noSpecName := configSupport.lookupConfigSpec("missing")
			Expect(noSpecName).To(BeNil())
		})
		FreshPluginAssertions(&input)

		Context("Add a creation goal", func() {
			CreatedGoalAssertions(&input)

			Context("Manually reconcile goal", func() {
				ManuallyReconcileGoal(&input)

				Context("Manually store goal", func() {
					ManuallyStoreGoal(&input)
				})
				Context("Manually update goal", func() {
					ManuallyReconcileUpdateGoal(&input)
				})
				Context("Manually remove goal", func() {
					ManuallyRemoveGoal(&input)
					Context("Manually remove goal", func() {
						ManuallyReconcileRemoveGoal(&input)
					})
				})
			})
		})
	})
})

var _ = Describe("Algorithm Plugin for Secret - Stack Request", func() {
	var (
		secretInit    initializationSecret
		stack         types.Stack
		input         AlgorithmPluginInputs
		err1, err2    error
		snapshot      interfaces.SnapshotStack
		secretSupport algorithmSecret
	)
	BeforeEach(func() {
		input.cli = fakes.NewFakeReconcilerClient()
		secretInit = newInitializationSupportSecret(input.cli)
		stack = fakes.GetTestStack("stack12")
		input.stack = &stack
		input.algorithmInit = secretInit
		input.stackID, err1 = input.cli.AddStack(input.stack.Spec)
		snapshot, err2 = input.cli.FakeStackStore.GetSnapshotStack(input.stackID)
		input.specName = input.stack.Spec.Secrets[0].Annotations.Name
	})
	It("Initializations Succeed", func() {
		Expect(err1).ToNot(HaveOccurred())
		Expect(err2).ToNot(HaveOccurred())
	})
	BeforeEach(func() {
		input.request = &interfaces.ReconcileResource{
			SnapshotResource: interfaces.SnapshotResource{
				ID: input.stackID,
			},
			Kind: interfaces.ReconcileStack,
		}

		secretSupport := newAlgorithmPluginSecret(secretInit, snapshot, input.request)
		input.resourcePlugin = secretSupport
	})
	Context("With a fresh plugin", func() {
		It("Coverage missing lookup", func() {
			noSpecName := secretSupport.lookupSecretSpec("missing")
			Expect(noSpecName).To(BeNil())
		})
		FreshPluginAssertions(&input)

		Context("Add a creation goal", func() {
			CreatedGoalAssertions(&input)

			Context("Manually reconcile goal", func() {
				ManuallyReconcileGoal(&input)

				Context("Manually store goal", func() {
					ManuallyStoreGoal(&input)
					Context("Manually update goal", func() {
						BeforeEach(func() {
							snapshot, err2 = input.cli.FakeStackStore.GetSnapshotStack(input.stackID)
							secretSupport := newAlgorithmPluginSecret(secretInit, snapshot, input.request)
							input.resourcePlugin = secretSupport
							input.resource = secretSupport.getGoalResource(input.resource.Name)
							act, _ := secretSupport.getActiveResource(*input.resource)
							input.resource.Meta = act.getSnapshot().Meta
							Expect(input.resource).ToNot(BeNil())
						})
						ManuallyReconcileUpdateGoal(&input)
					})
				})

				Context("Manually remove goal", func() {
					ManuallyRemoveGoal(&input)
					Context("Manually remove goal", func() {
						ManuallyReconcileRemoveGoal(&input)
					})
				})
			})
		})
	})
})

var _ = Describe("Algorithm Plugin for Network - Stack Request", func() {
	var (
		networkInit    initializationNetwork
		stack          types.Stack
		input          AlgorithmPluginInputs
		err1, err2     error
		snapshot       interfaces.SnapshotStack
		networkSupport algorithmNetwork
	)
	BeforeEach(func() {
		input.cli = fakes.NewFakeReconcilerClient()
		networkInit = newInitializationSupportNetwork(input.cli)
		stack = fakes.GetTestStack("stack12")
		input.stack = &stack
		input.algorithmInit = networkInit
		input.stackID, err1 = input.cli.AddStack(input.stack.Spec)
		snapshot, err2 = input.cli.FakeStackStore.GetSnapshotStack(input.stackID)
		for networkName := range input.stack.Spec.Networks {
			input.specName = networkName
			break
		}
	})
	It("Initializations Succeed", func() {
		Expect(err1).ToNot(HaveOccurred())
		Expect(err2).ToNot(HaveOccurred())
	})
	BeforeEach(func() {
		input.request = &interfaces.ReconcileResource{
			SnapshotResource: interfaces.SnapshotResource{
				ID: input.stackID,
			},
			Kind: interfaces.ReconcileStack,
		}

		networkSupport := newAlgorithmPluginNetwork(networkInit, snapshot, input.request)
		input.resourcePlugin = networkSupport
	})
	Context("With a fresh plugin", func() {
		It("Coverage missing lookup", func() {
			noSpecName := networkSupport.lookupNetworkSpec("missing")
			Expect(noSpecName).To(BeNil())
		})
		FreshPluginAssertions(&input)

		Context("Add a creation goal", func() {
			CreatedGoalAssertions(&input)

			Context("Manually reconcile goal", func() {
				ManuallyReconcileGoal(&input)

				Context("Manually store goal", func() {
					ManuallyStoreGoal(&input)
				})
				Context("Manually update goal", func() {
					BeforeEach(func() {
						networkSupport := newAlgorithmPluginNetwork(networkInit, snapshot, input.request)
						input.resourcePlugin = networkSupport
					})
					ManuallyReconcileUpdateGoal(&input)
				})
				Context("Manually remove goal", func() {
					ManuallyRemoveGoal(&input)
					Context("Manually remove goal", func() {
						ManuallyReconcileRemoveGoal(&input)
					})
				})
			})
		})
	})
})
