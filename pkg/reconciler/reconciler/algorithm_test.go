package reconciler

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/types"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/errdefs"

	"github.com/docker/stacks/pkg/fakes"
	"github.com/docker/stacks/pkg/interfaces"
	"github.com/docker/stacks/pkg/reconciler/notifier"
	"github.com/docker/stacks/pkg/types"
)

func ConsistOfStates(states ...interfaces.ReconcileState) GomegaMatcher {
	state := func(goals []*interfaces.ReconcileResource) []interfaces.ReconcileState {
		result := []interfaces.ReconcileState{}
		for _, v := range goals {
			result = append(result, v.Mark)
		}
		return result
	}
	other := []interface{}{}
	for _, s := range states {
		other = append(other, s)
	}
	return WithTransform(state, ConsistOf(other...))
}
func ConsistOfIDs(ids ...string) GomegaMatcher {
	id := func(goals []*interfaces.ReconcileResource) []string {
		result := []string{}
		for _, v := range goals {
			result = append(result, v.ID)
		}
		return result
	}
	other := []interface{}{}
	for _, s := range ids {
		other = append(other, s)
	}
	return WithTransform(id, ConsistOf(other...))
}

// addSpecifiedResources uses the plugin implementations in order to use the abstracted interfaces
// so that a generalized test possible
func addSpecifiedResources(init initializationSupport, stackSpec types.StackSpec, stackID string) []string {
	snapshot := interfaces.SnapshotStack{
		SnapshotResource: interfaces.SnapshotResource{
			ID: stackID,
		},
		CurrentSpec: stackSpec,
	}
	emptyRequest := &interfaces.ReconcileResource{}
	plugin := init.createPlugin(snapshot, emptyRequest)

	names := plugin.getSpecifiedResourceNames()
	Expect(names).To(HaveLen(2))
	resource1 := plugin.addCreateResourceGoal(names[0])
	resource2 := plugin.addCreateResourceGoal(names[1])
	err1 := plugin.createResource(resource1)
	err2 := plugin.createResource(resource2)

	Expect(err1).ToNot(HaveOccurred())
	Expect(err2).ToNot(HaveOccurred())

	return []string{resource1.ID, resource2.ID}
}

func removeSpecifiedResource(init initializationSupport, snapshot interfaces.SnapshotStack, name string, stackID string) error {
	emptyRequest := &interfaces.ReconcileResource{}
	plugin := init.createPlugin(snapshot, emptyRequest)
	resource := plugin.getGoalResource(name)
	active, err := plugin.getActiveResource(*resource)
	Expect(err).ToNot(HaveOccurred())
	resource.ID = active.getSnapshot().ID
	return plugin.deleteResource(resource)
}

func changeSpecifiedResource(init initializationSupport, snapshot interfaces.SnapshotStack, name string, stackID string) error {
	emptyRequest := &interfaces.ReconcileResource{}
	plugin := init.createPlugin(snapshot, emptyRequest)
	resource := plugin.getGoalResource(name)
	active, err := plugin.getActiveResource(*resource)
	Expect(err).ToNot(HaveOccurred())
	resource.ID = active.getSnapshot().ID
	resource.Meta = active.getSnapshot().Meta
	if init.getKind() == interfaces.ReconcileService {
		resource.Config.(*swarm.ServiceSpec).UpdateConfig = &swarm.UpdateConfig{}
	} else if init.getKind() == interfaces.ReconcileConfig {
		resource.Config.(*swarm.ConfigSpec).Templating = &swarm.Driver{}
	} else if init.getKind() == interfaces.ReconcileSecret {
		resource.Config.(*swarm.SecretSpec).Driver = &swarm.Driver{}
	} else if init.getKind() == interfaces.ReconcileNetwork {
		resource.Config.(*dockerTypes.NetworkCreateRequest).NetworkCreate.Driver = "driver"
	}
	err1 := plugin.updateResource(*resource)
	return err1
}

func mutateSnapshot(init initializationSupport, snapshot *interfaces.SnapshotStack) {
	if init.getKind() == interfaces.ReconcileService {
		service := snapshot.Services[0]
		snapshot.Services = []interfaces.SnapshotResource{service}
	} else if init.getKind() == interfaces.ReconcileConfig {
		config := snapshot.Configs[0]
		snapshot.Configs = []interfaces.SnapshotResource{config}
	} else if init.getKind() == interfaces.ReconcileSecret {
		secret := snapshot.Secrets[0]
		snapshot.Secrets = []interfaces.SnapshotResource{secret}
	} else if init.getKind() == interfaces.ReconcileNetwork {
		network := snapshot.Networks[0]
		snapshot.Networks = []interfaces.SnapshotResource{network}
	}
}

func mutateStackSpec(init initializationSupport, stackSpec *types.StackSpec) {
	if init.getKind() == interfaces.ReconcileService {
		service := stackSpec.Services[0]
		stackSpec.Services = []swarm.ServiceSpec{service}
	} else if init.getKind() == interfaces.ReconcileConfig {
		config := stackSpec.Configs[0]
		stackSpec.Configs = []swarm.ConfigSpec{config}
	} else if init.getKind() == interfaces.ReconcileSecret {
		secret := stackSpec.Secrets[0]
		stackSpec.Secrets = []swarm.SecretSpec{secret}
	} else if init.getKind() == interfaces.ReconcileNetwork {
		for k := range stackSpec.Networks {
			delete(stackSpec.Networks, k)
			break
		}
	}
}

type Stuff struct {
	cli                            *fakes.FakeReconcilerClient
	pluginInit                     initializationSupport
	stackSpec, plainStackSpec      types.StackSpec
	cheatStackSpec, errorStackSpec types.StackSpec
	plainStackSpecRemoveErr        types.StackSpec
	plainStackSpecUpdateErr        types.StackSpec
	plainStackSpecGetNetworksErr   types.StackSpec
	plainStackSpecGetServicesErr   types.StackSpec
	plainStackSpecGetSecretsErr    types.StackSpec
	plainStackSpecGetConfigsErr    types.StackSpec
	stackID                        string
	request                        *interfaces.ReconcileResource
	alternateStackIDs              map[string]string
}

func makeStuff(stuff *Stuff) {
	stuff.cli = fakes.NewFakeReconcilerClient()

	stuff.cli.FakeStackStore.SpecifyErrorTrigger("SpecifiedError", fakes.FakeUnimplemented)
	stuff.cli.FakeServiceStore.SpecifyErrorTrigger("SpecifiedError", fakes.FakeUnimplemented)
	stuff.cli.FakeSecretStore.SpecifyErrorTrigger("SpecifiedError", fakes.FakeUnimplemented)
	stuff.cli.FakeConfigStore.SpecifyErrorTrigger("SpecifiedError", fakes.FakeUnimplemented)
	stuff.cli.FakeNetworkStore.SpecifyErrorTrigger("SpecifiedError", fakes.FakeUnimplemented)

	stuff.plainStackSpecRemoveErr = fakes.GetTestStackSpecWithMultipleSpecs(2, "AlgoTest")
	stuff.plainStackSpecUpdateErr = fakes.GetTestStackSpecWithMultipleSpecs(2, "AlgoTest")
	stuff.plainStackSpec = fakes.GetTestStackSpecWithMultipleSpecs(2, "AlgoTest")
	stuff.stackSpec = fakes.GetTestStackSpecWithMultipleSpecs(2, "AlgoTest")
	stuff.errorStackSpec = fakes.GetTestStackSpecWithMultipleSpecs(2, "AlgoTest")
	stuff.cheatStackSpec = fakes.GetTestStackSpecWithMultipleSpecs(2, "AlgoTestCheat")

	/* BEGIN A few steps are used to populate the following types.StackSpecs */
	stuff.plainStackSpecGetNetworksErr = fakes.GetTestStackSpecWithMultipleSpecs(2, "AlgoTestNetworks")
	stuff.plainStackSpecGetServicesErr = fakes.GetTestStackSpecWithMultipleSpecs(2, "AlgoTestServices")
	stuff.plainStackSpecGetSecretsErr = fakes.GetTestStackSpecWithMultipleSpecs(2, "AlgoTestSecrets")
	stuff.plainStackSpecGetConfigsErr = fakes.GetTestStackSpecWithMultipleSpecs(2, "AlgoTestConfigs")

	plainStackSpecGetResourcesErr := fakes.GetTestStackSpecWithMultipleSpecs(2, "AlgoTest")
	fakes.InjectForcedGetResourcesError(stuff.cli, &plainStackSpecGetResourcesErr, "SpecifiedError")

	stuff.plainStackSpecGetNetworksErr.Networks = plainStackSpecGetResourcesErr.Networks
	stuff.plainStackSpecGetServicesErr.Services = plainStackSpecGetResourcesErr.Services
	stuff.plainStackSpecGetSecretsErr.Secrets = plainStackSpecGetResourcesErr.Secrets
	stuff.plainStackSpecGetConfigsErr.Configs = plainStackSpecGetResourcesErr.Configs
	/* END */

	stuff.alternateStackIDs = map[string]string{}
	stuff.stackID, _ = stuff.cli.AddStack(stuff.plainStackSpec)
	for _, ss := range []types.StackSpec{
		stuff.plainStackSpecGetNetworksErr,
		stuff.plainStackSpecGetServicesErr,
		stuff.plainStackSpecGetSecretsErr,
		stuff.plainStackSpecGetConfigsErr,
	} {
		id, err := stuff.cli.AddStack(ss)
		stuff.alternateStackIDs[ss.Annotations.Name] = id
		Expect(err).ToNot(HaveOccurred())
	}

	fakes.InjectForcedRemoveError(stuff.cli, &stuff.plainStackSpecRemoveErr, "SpecifiedError")
	fakes.InjectForcedUpdateError(stuff.cli, &stuff.plainStackSpecUpdateErr, "SpecifiedError")
	fakes.InjectForcedGetError(stuff.cli, &stuff.errorStackSpec, "SpecifiedError")

	fakes.InjectStackID(&stuff.stackSpec, stuff.stackID)
	fakes.InjectStackID(&stuff.errorStackSpec, stuff.stackID)
	fakes.InjectStackID(&stuff.cheatStackSpec, stuff.stackID)

	Expect(stuff.stackSpec.Networks).To(HaveLen(2))
	Expect(stuff.stackSpec.Services).To(HaveLen(2))
	Expect(stuff.stackSpec.Secrets).To(HaveLen(2))
	Expect(stuff.stackSpec.Configs).To(HaveLen(2))
}

func algorithmTest(stuff *Stuff) {
	It("Precondition", func() {
		Expect(stuff).ToNot(BeNil())
	})
	var (
		algorithmPlugin   algorithmPlugin
		err, err1         error
		resourceIDs       []string
		current, snapshot interfaces.SnapshotStack
	)
	Context("Creating the expected resources beforehand", func() {
		BeforeEach(func() {
			resourceIDs = addSpecifiedResources(stuff.pluginInit, stuff.stackSpec, stuff.stackID)
			snapshot, _ = stuff.cli.FakeStackStore.GetSnapshotStack(stuff.stackID)
			algorithmPlugin = stuff.pluginInit.createPlugin(snapshot, stuff.request)
			current, err = algorithmPlugin.reconcile(snapshot)
		})
		It("New resources created", func() {
			Expect(algorithmPlugin.getGoalResources()).To(ConsistOfStates(interfaces.ReconcileCreate, interfaces.ReconcileCreate))
			Expect(algorithmPlugin.getGoalResources()).ToNot(ConsistOfIDs(resourceIDs[0], resourceIDs[1]))
			Expect(algorithmPlugin.getSpecifiedResourceNames()).To(HaveLen(2))

		})
	})
	Context("Creating unexpected but labelled resources beforehand", func() {
		BeforeEach(func() {
			resourceIDs = addSpecifiedResources(stuff.pluginInit, stuff.cheatStackSpec, stuff.stackID)
			snapshot, _ = stuff.cli.FakeStackStore.GetSnapshotStack(stuff.stackID)
			algorithmPlugin = stuff.pluginInit.createPlugin(snapshot, stuff.request)
			current, err = algorithmPlugin.reconcile(snapshot)
		})
		It("New resources created", func() {
			Expect(resourceIDs).To(HaveLen(2))
			Expect(err).ToNot(HaveOccurred())
			Expect(algorithmPlugin.getGoalResources()).To(ConsistOfStates(interfaces.ReconcileCreate, interfaces.ReconcileCreate, interfaces.ReconcileDelete, interfaces.ReconcileDelete))
		})
	})
	Context("Creating matching but unlabelled resources beforehand", func() {
		BeforeEach(func() {
			resourceIDs = addSpecifiedResources(stuff.pluginInit, stuff.plainStackSpec, "")
			snapshot, _ := stuff.cli.FakeStackStore.GetSnapshotStack(stuff.stackID)
			algorithmPlugin = stuff.pluginInit.createPlugin(snapshot, stuff.request)
			current, err = algorithmPlugin.reconcile(snapshot)
		})
		It("they should not fail", func() {
			Expect(algorithmPlugin.getSpecifiedResourceNames()).To(HaveLen(2))
			Expect(current).ToNot(BeNil())
			Expect(errdefs.IsAlreadyExists(err)).To(BeTrue())
		})
	})
	Context("Unmodified creation reconciliation with RemoveError config", func() {
		BeforeEach(func() {
			snapshot, _ := stuff.cli.FakeStackStore.GetSnapshotStack(stuff.stackID)
			_ = stuff.cli.UpdateStack(stuff.stackID, stuff.plainStackSpecRemoveErr, snapshot.Meta.Version.Index)
			snapshot, _ = stuff.cli.FakeStackStore.GetSnapshotStack(stuff.stackID)
			algorithmPlugin = stuff.pluginInit.createPlugin(snapshot, stuff.request)
			current, err = algorithmPlugin.reconcile(snapshot)
		})
		It("they should not fail", func() {
			Expect(algorithmPlugin.getSpecifiedResourceNames()).To(HaveLen(2))
			Expect(current).ToNot(BeNil())
			Expect(stuff.pluginInit.getSnapshotResourceNames(current)).To(HaveLen(2))
			Expect(err).ToNot(HaveOccurred())
			Expect(current.Meta.Version.Index).To(Equal(uint64(5)))
		})
		Context("Remove resource using stack reconciliation cause error", func() {
			var altered interfaces.SnapshotStack
			BeforeEach(func() {
				mutateStackSpec(stuff.pluginInit, &stuff.plainStackSpecRemoveErr)
				err1 = stuff.cli.UpdateStack(stuff.stackID, stuff.plainStackSpecRemoveErr, current.Meta.Version.Index)
				altered, _ = stuff.cli.FakeStackStore.GetSnapshotStack(stuff.stackID)

				algorithmPlugin = stuff.pluginInit.createPlugin(altered, stuff.request)
				_, err = algorithmPlugin.reconcile(altered)
			})
			It("All should be same", func() {
				Expect(err == fakes.FakeUnimplemented).To(BeTrue())
				Expect(err1).ToNot(HaveOccurred())
			})
		})
	})
	Context("Unmodified creation reconciliation with UpdateError config", func() {
		BeforeEach(func() {
			snapshot, _ := stuff.cli.FakeStackStore.GetSnapshotStack(stuff.stackID)
			_ = stuff.cli.UpdateStack(stuff.stackID, stuff.plainStackSpecUpdateErr, snapshot.Meta.Version.Index)
			snapshot, _ = stuff.cli.FakeStackStore.GetSnapshotStack(stuff.stackID)
			algorithmPlugin = stuff.pluginInit.createPlugin(snapshot, stuff.request)
			current, err = algorithmPlugin.reconcile(snapshot)
		})
		It("they should not fail", func() {
			Expect(algorithmPlugin.getSpecifiedResourceNames()).To(HaveLen(2))
			Expect(current).ToNot(BeNil())
			Expect(stuff.pluginInit.getSnapshotResourceNames(current)).To(HaveLen(2))
			Expect(err).ToNot(HaveOccurred())
			Expect(current.Meta.Version.Index).To(Equal(uint64(5)))
		})
		Context("Remove resource using stack reconciliation cause error", func() {
			var altered interfaces.SnapshotStack
			BeforeEach(func() {
				mutateStackSpec(stuff.pluginInit, &stuff.plainStackSpecUpdateErr)
				err1 = stuff.cli.UpdateStack(stuff.stackID, stuff.plainStackSpecUpdateErr, current.Meta.Version.Index)
				altered, _ = stuff.cli.FakeStackStore.GetSnapshotStack(stuff.stackID)

				algorithmPlugin = stuff.pluginInit.createPlugin(altered, stuff.request)
				_, err = algorithmPlugin.reconcile(altered)
			})
			It("All should be same", func() {
				if algorithmPlugin.getKind() == interfaces.ReconcileNetwork {
					Skip("NETWORKS DO NOT HAVE UPDATE AVAILABLE")
				}
				Expect(err == fakes.FakeUnimplemented).To(BeTrue())
				Expect(err1).ToNot(HaveOccurred())
			})
		})
	})
	Context("Unmodified creation reconciliation", func() {
		BeforeEach(func() {
			snapshot, _ := stuff.cli.FakeStackStore.GetSnapshotStack(stuff.stackID)
			algorithmPlugin = stuff.pluginInit.createPlugin(snapshot, stuff.request)
			current, err = algorithmPlugin.reconcile(snapshot)
		})
		It("they should not fail", func() {
			Expect(algorithmPlugin.getSpecifiedResourceNames()).To(HaveLen(2))
			Expect(current).ToNot(BeNil())
			Expect(stuff.pluginInit.getSnapshotResourceNames(current)).To(HaveLen(2))
			Expect(err).ToNot(HaveOccurred())
			Expect(current.Meta.Version.Index).To(Equal(uint64(4)))
		})
		Context("Fast-path NOP reconciliation", func() {
			var deeper interfaces.SnapshotStack
			BeforeEach(func() {
				algorithmPlugin = stuff.pluginInit.createPlugin(current, stuff.request)
				deeper, err = algorithmPlugin.reconcile(current)
			})
			It("All should be same", func() {
				Expect(algorithmPlugin.getGoalResources()).To(ConsistOfStates(interfaces.ReconcileSame, interfaces.ReconcileSame))
				Expect(deeper).ToNot(BeNil())
				Expect(stuff.pluginInit.getSnapshotResourceNames(deeper)).To(HaveLen(2))
				Expect(deeper.ID).To(Equal(stuff.stackID))
			})
		})
		Context("Fast-path NOP reconciliation With SKIP", func() {
			var deeper interfaces.SnapshotStack
			BeforeEach(func() {
				algorithmPlugin = stuff.pluginInit.createPlugin(current, stuff.request)
				name := algorithmPlugin.getSpecifiedResourceNames()[0]
				algorithmPlugin.getGoalResource(name).Mark = interfaces.ReconcileSkip
				deeper, err = algorithmPlugin.reconcile(current)
			})
			It("All should be same", func() {
				Expect(algorithmPlugin.getGoalResources()).To(ConsistOfStates(interfaces.ReconcileSkip, interfaces.ReconcileSame))
				Expect(deeper).ToNot(BeNil())
				Expect(stuff.pluginInit.getSnapshotResourceNames(deeper)).To(HaveLen(2))
				Expect(deeper.ID).To(Equal(stuff.stackID))
			})
		})
		Context("Remove resource using stack reconciliation", func() {
			var deeper, altered interfaces.SnapshotStack
			BeforeEach(func() {
				mutateStackSpec(stuff.pluginInit, &stuff.plainStackSpec)
				err1 = stuff.cli.UpdateStack(stuff.stackID, stuff.plainStackSpec, current.Meta.Version.Index)
				altered, _ = stuff.cli.FakeStackStore.GetSnapshotStack(stuff.stackID)

				algorithmPlugin = stuff.pluginInit.createPlugin(altered, stuff.request)
				deeper, err = algorithmPlugin.reconcile(altered)
			})
			It("All should be same", func() {
				Expect(algorithmPlugin.getSpecifiedResourceNames()).To(HaveLen(1))
				Expect(err1).ToNot(HaveOccurred())
				Expect(algorithmPlugin.getGoalResources()).To(ConsistOfStates(interfaces.ReconcileSame, interfaces.ReconcileDelete))
				Expect(deeper).ToNot(BeNil())
				Expect(stuff.pluginInit.getSnapshotResourceNames(deeper)).To(HaveLen(1))
			})
		})
		Context("Remove resource before stack reconciliation", func() {
			var deeper interfaces.SnapshotStack
			BeforeEach(func() {
				algorithmPlugin = stuff.pluginInit.createPlugin(current, stuff.request)
				name := algorithmPlugin.getSpecifiedResourceNames()[0]
				err1 = removeSpecifiedResource(stuff.pluginInit, current, name, stuff.stackID)
				deeper, err = algorithmPlugin.reconcile(current)
			})
			It("All should be same", func() {
				Expect(err1).ToNot(HaveOccurred())
				Expect(err).ToNot(HaveOccurred())
				Expect(algorithmPlugin.getGoalResources()).To(ConsistOfStates(interfaces.ReconcileSame, interfaces.ReconcileCreate))
				Expect(deeper).ToNot(BeNil())
				Expect(stuff.pluginInit.getSnapshotResourceNames(deeper)).To(HaveLen(2))
			})
		})
		Context("Remove snapshot info from stack before stack reconciliation", func() {
			var deeper interfaces.SnapshotStack
			BeforeEach(func() {
				mutateSnapshot(stuff.pluginInit, &current)
				deeper, err1 = stuff.cli.UpdateSnapshotStack(stuff.stackID, current, current.Meta.Version.Index)
				Expect(stuff.pluginInit.getSnapshotResourceNames(deeper)).To(HaveLen(1))

				algorithmPlugin = stuff.pluginInit.createPlugin(deeper, stuff.request)
				deeper, err = algorithmPlugin.reconcile(deeper)
			})
			It("All should be same", func() {
				Expect(err1).ToNot(HaveOccurred())
				Expect(err).ToNot(HaveOccurred())
				Expect(algorithmPlugin.getGoalResources()).To(ConsistOfStates(interfaces.ReconcileSame, interfaces.ReconcileCreate))
				Expect(deeper).ToNot(BeNil())
				Expect(stuff.pluginInit.getSnapshotResourceNames(deeper)).To(HaveLen(2))
			})
		})
		Context("Remove resource and Increment snapshot version before stack reconciliation", func() {
			BeforeEach(func() {
				name := algorithmPlugin.getSpecifiedResourceNames()[0]
				err1 = removeSpecifiedResource(stuff.pluginInit, current, name, stuff.stackID)

				_, err1 = stuff.cli.UpdateSnapshotStack(stuff.stackID, current, current.Meta.Version.Index)
				algorithmPlugin = stuff.pluginInit.createPlugin(current, stuff.request)
				_, err = algorithmPlugin.reconcile(current)
			})
			It("All should be same", func() {
				Expect(err1).ToNot(HaveOccurred())
				Expect(err).To(HaveOccurred())
			})
		})
		Context("Change resource before stack reconciliation", func() {
			var deeper interfaces.SnapshotStack
			BeforeEach(func() {
				name := algorithmPlugin.getSpecifiedResourceNames()[0]
				err1 = changeSpecifiedResource(stuff.pluginInit, current, name, stuff.stackID)
				algorithmPlugin = stuff.pluginInit.createPlugin(current, stuff.request)
				deeper, err = algorithmPlugin.reconcile(current)
			})
			It("All should be same", func() {
				if algorithmPlugin.getKind() == interfaces.ReconcileNetwork {
					Skip("NETWORKS DO NOT HAVE UPDATE AVAILABLE")
				}
				Expect(err1).ToNot(HaveOccurred())
				Expect(err).ToNot(HaveOccurred())
				Expect(algorithmPlugin.getGoalResources()).To(ConsistOfStates(interfaces.ReconcileSame, interfaces.ReconcileUpdate))
				Expect(deeper).ToNot(BeNil())
				Expect(stuff.pluginInit.getSnapshotResourceNames(deeper)).To(HaveLen(2))
			})
		})
	})
}

var _ = Describe("Algorithm Test - Services", func() {
	var (
		stuff Stuff
	)
	BeforeEach(func() {
		makeStuff(&stuff)
		stuff.pluginInit = newInitializationSupportService(stuff.cli)
		stuff.request = &interfaces.ReconcileResource{
			SnapshotResource: interfaces.SnapshotResource{
				ID: stuff.stackID,
			},
			Kind: interfaces.ReconcileStack,
		}
	})
	Context("Share Test", func() {
		algorithmTest(&stuff)
	})
})

var _ = Describe("Algorithm Test - Configs", func() {
	var (
		stuff Stuff
	)
	BeforeEach(func() {
		makeStuff(&stuff)
		stuff.pluginInit = newInitializationSupportConfig(stuff.cli)
		stuff.request = &interfaces.ReconcileResource{
			SnapshotResource: interfaces.SnapshotResource{
				ID: stuff.stackID,
			},
			Kind: interfaces.ReconcileStack,
		}
	})
	algorithmTest(&stuff)
})

var _ = Describe("Algorithm Test - Secrets", func() {
	var (
		stuff Stuff
	)
	BeforeEach(func() {
		makeStuff(&stuff)
		stuff.pluginInit = newInitializationSupportSecret(stuff.cli)
		stuff.request = &interfaces.ReconcileResource{
			SnapshotResource: interfaces.SnapshotResource{
				ID: stuff.stackID,
			},
			Kind: interfaces.ReconcileStack,
		}
	})
	algorithmTest(&stuff)
})

var _ = Describe("Algorithm Test - Networks", func() {
	var (
		stuff Stuff
	)
	BeforeEach(func() {
		makeStuff(&stuff)
		stuff.pluginInit = newInitializationSupportNetwork(stuff.cli)
		stuff.request = &interfaces.ReconcileResource{
			SnapshotResource: interfaces.SnapshotResource{
				ID: stuff.stackID,
			},
			Kind: interfaces.ReconcileStack,
		}
	})
	algorithmTest(&stuff)
})

var _ = Describe("Algorithm Test - Whole Stack", func() {
	var (
		stuff Stuff
	)
	BeforeEach(func() {
		makeStuff(&stuff)
		stuff.request = &interfaces.ReconcileResource{
			SnapshotResource: interfaces.SnapshotResource{
				ID: stuff.stackID,
			},
			Kind: interfaces.ReconcileStack,
		}
	})
	Context("Reconciling the expected get resources failures into existenence", func() {
		var (
			err        error
			reconciler *reconciler
		)
		BeforeEach(func() {
			x := notifier.NewNotificationForwarder()
			reconciler = newReconciler(x, stuff.cli)
			for _, v := range stuff.alternateStackIDs {
				stuff.request = &interfaces.ReconcileResource{
					SnapshotResource: interfaces.SnapshotResource{
						ID: v,
					},
					Kind: interfaces.ReconcileStack,
				}
				err = reconciler.Reconcile(stuff.request)
				Expect(err).ToNot(HaveOccurred())
				err = reconciler.Reconcile(stuff.request)
			}
		})
		It("New resources created", func() {
			Expect(err).To(HaveOccurred())
			Expect(err == fakes.FakeUnimplemented).To(BeTrue())
		})
	})
	Context("Reconciling the expected resources into existenence", func() {
		var (
			snapshot   interfaces.SnapshotStack
			err        error
			reconciler *reconciler
		)
		BeforeEach(func() {
			x := notifier.NewNotificationForwarder()
			reconciler = newReconciler(x, stuff.cli)
			err = reconciler.Reconcile(stuff.request)
			snapshot, _ = stuff.cli.FakeStackStore.GetSnapshotStack(stuff.stackID)
		})
		It("New resources created", func() {
			Expect(err).ToNot(HaveOccurred())
			//
			// The number of versions indiciates the first store
			// the 4 algorithmPlugins making 3 storeGoals calls each
			// (1 for the algorithm and 2 for the specs)
			// 1 + 4 (1 + 2) == 13
			//
			Expect(snapshot.Meta.Version.Index).To(Equal(uint64(13)))
			//
			// These lengths match the type.StackSpec.  We rely
			// on the fakes.* tests for testing the details
			//
			Expect(snapshot.Networks).To(HaveLen(2))
			Expect(snapshot.Services).To(HaveLen(2))
			Expect(snapshot.Secrets).To(HaveLen(2))
			Expect(snapshot.Configs).To(HaveLen(2))

			Expect(reconciler.getRequestedResource()).To(Equal(stuff.request))
		})
		Context("Reconciling subresource - Service", func() {
			BeforeEach(func() {
				stuff.request = &interfaces.ReconcileResource{
					SnapshotResource: interfaces.SnapshotResource{
						ID:   snapshot.Services[0].ID,
						Name: snapshot.Services[0].Name,
					},
					Kind: interfaces.ReconcileService,
				}
				err = reconciler.Reconcile(stuff.request)
				snapshot, _ = stuff.cli.FakeStackStore.GetSnapshotStack(stuff.stackID)
			})
			It("Skipping all but one unchanged resource", func() {
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Context("Reconciling subresource - Secret", func() {
			BeforeEach(func() {
				stuff.request = &interfaces.ReconcileResource{
					SnapshotResource: interfaces.SnapshotResource{
						ID:   snapshot.Secrets[0].ID,
						Name: snapshot.Secrets[0].Name,
					},
					Kind: interfaces.ReconcileSecret,
				}
				err = reconciler.Reconcile(stuff.request)
				snapshot, _ = stuff.cli.FakeStackStore.GetSnapshotStack(stuff.stackID)
			})
			It("Skipping all but one unchanged resource", func() {
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Context("Reconciling subresource - Config", func() {
			BeforeEach(func() {
				stuff.request = &interfaces.ReconcileResource{
					SnapshotResource: interfaces.SnapshotResource{
						ID:   snapshot.Configs[0].ID,
						Name: snapshot.Configs[0].Name,
					},
					Kind: interfaces.ReconcileConfig,
				}
				err = reconciler.Reconcile(stuff.request)
				snapshot, _ = stuff.cli.FakeStackStore.GetSnapshotStack(stuff.stackID)
			})
			It("Skipping all but one unchanged resource", func() {
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Context("Reconciling subresource - Network", func() {
			BeforeEach(func() {
				stuff.request = &interfaces.ReconcileResource{
					SnapshotResource: interfaces.SnapshotResource{
						ID:   snapshot.Networks[0].ID,
						Name: snapshot.Networks[0].Name,
					},
					Kind: interfaces.ReconcileNetwork,
				}
				err = reconciler.Reconcile(stuff.request)
				snapshot, _ = stuff.cli.FakeStackStore.GetSnapshotStack(stuff.stackID)
			})
			It("Skipping all but one unchanged resource", func() {
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Context("Reconciling Stack with forced error - Network", func() {
			BeforeEach(func() {
				stack, _ := stuff.cli.GetStack(stuff.stackID)
				stuff.cli.FakeStackStore.MarkStackSpecForError("SpecifiedError", &stack.Spec, "GetSnapshotStack")
				err1 := stuff.cli.UpdateStack(
					stuff.stackID,
					stack.Spec,
					stack.Meta.Version.Index)
				Expect(err1).ToNot(HaveOccurred())

				stuff.request = &interfaces.ReconcileResource{
					SnapshotResource: interfaces.SnapshotResource{
						ID: stuff.stackID,
					},
					Kind: interfaces.ReconcileStack,
				}

				err = reconciler.Reconcile(stuff.request)
			})
			It("Skipping all but one unchanged resource", func() {
				Expect(err).To(HaveOccurred())
				Expect(err == fakes.FakeUnimplemented).To(BeTrue())
			})
		})
		Context("Reconciling non-existent subresource - Service", func() {
			BeforeEach(func() {
				stuff.request = &interfaces.ReconcileResource{
					SnapshotResource: interfaces.SnapshotResource{
						ID:   "MISSING",
						Name: snapshot.Services[0].Name,
					},
					Kind: interfaces.ReconcileService,
				}
				err = reconciler.Reconcile(stuff.request)
				snapshot, _ = stuff.cli.FakeStackStore.GetSnapshotStack(stuff.stackID)
			})
			It("Nothing to do, no required error", func() {
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Context("Reconciling unlabeled subresource - Config", func() {
			BeforeEach(func() {
				resource := snapshot.Configs[0]
				cfg, _ := stuff.cli.GetConfig(resource.ID)
				delete(cfg.Spec.Annotations.Labels, types.StackLabel)
				errUpdate := stuff.cli.UpdateConfig(
					resource.ID,
					cfg.Meta.Version.Index,
					cfg.Spec)
				Expect(errUpdate).ToNot(HaveOccurred())

				stuff.request = &interfaces.ReconcileResource{
					SnapshotResource: interfaces.SnapshotResource{
						ID:   snapshot.Configs[0].ID,
						Name: snapshot.Configs[0].Name,
					},
					Kind: interfaces.ReconcileConfig,
				}
				err = reconciler.Reconcile(stuff.request)
				snapshot, _ = stuff.cli.FakeStackStore.GetSnapshotStack(stuff.stackID)
			})
			It("Skipping all but one unchanged resource", func() {
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Context("Reconciling subresource with embedded error - Config", func() {
			BeforeEach(func() {
				resource := snapshot.Configs[0]
				cfg, _ := stuff.cli.GetConfig(resource.ID)
				stuff.cli.FakeConfigStore.MarkConfigSpecForError("SpecifiedError", &cfg.Spec, "GetConfig")
				errUpdate := stuff.cli.UpdateConfig(
					resource.ID,
					cfg.Meta.Version.Index,
					cfg.Spec)
				Expect(errUpdate).ToNot(HaveOccurred())

				stuff.request = &interfaces.ReconcileResource{
					SnapshotResource: interfaces.SnapshotResource{
						ID:   snapshot.Configs[0].ID,
						Name: snapshot.Configs[0].Name,
					},
					Kind: interfaces.ReconcileConfig,
				}
				err = reconciler.Reconcile(stuff.request)
				snapshot, _ = stuff.cli.FakeStackStore.GetSnapshotStack(stuff.stackID)
			})
			It("Reconcile curtailed by forced error", func() {
				Expect(err == fakes.FakeUnimplemented).To(BeTrue())
			})
		})
		Context("Reconciling mislabeled subresource - Secret", func() {
			BeforeEach(func() {
				resource := snapshot.Secrets[0]
				sct, _ := stuff.cli.GetSecret(resource.ID)
				sct.Spec.Annotations.Labels[types.StackLabel] = "MISSING"
				errUpdate := stuff.cli.UpdateSecret(
					resource.ID,
					sct.Meta.Version.Index,
					sct.Spec)
				Expect(errUpdate).ToNot(HaveOccurred())

				stuff.request = &interfaces.ReconcileResource{
					SnapshotResource: interfaces.SnapshotResource{
						ID:   snapshot.Secrets[0].ID,
						Name: snapshot.Secrets[0].Name,
					},
					Kind: interfaces.ReconcileSecret,
				}
				err = reconciler.Reconcile(stuff.request)
				snapshot, _ = stuff.cli.FakeStackStore.GetSnapshotStack(stuff.stackID)
			})
			It("Skipping all but one unchanged resource", func() {
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
