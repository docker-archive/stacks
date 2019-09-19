package reconciler

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/stacks/pkg/fakes"
	"github.com/docker/stacks/pkg/interfaces"
	"github.com/docker/stacks/pkg/types"
)

var _ = Describe("Initial Algorithm Support - Service", func() {
	var (
		cli         *fakes.FakeReconcilerClient
		serviceInit initializationService
		inputs      AlgorithmPluginInputs
		resources   []swarm.Service
		err         error
	)
	BeforeEach(func() {
		var resp1, resp2 *dockerTypes.ServiceCreateResponse
		cli = fakes.NewFakeReconcilerClient()
		serviceInit = newInitializationSupportService(cli)
		// testing two services one with a stacks label and
		// one without skipping zero-th element
		resources = fakes.GenerateServiceFixtures(3, "", "InitialService")
		resp1, inputs.err1 = cli.CreateService(resources[1].Spec,
			interfaces.DefaultCreateServiceArg2,
			interfaces.DefaultCreateServiceArg3)
		resp2, inputs.err2 = cli.CreateService(resources[2].Spec,
			interfaces.DefaultCreateServiceArg2,
			interfaces.DefaultCreateServiceArg3)
		inputs.algorithmInit = serviceInit
		inputs.search1.SnapshotResource = interfaces.SnapshotResource{
			Name: resources[1].Spec.Annotations.Name,
			ID:   resp1.ID,
		}
		inputs.search2.SnapshotResource = interfaces.SnapshotResource{
			Name: resources[2].Spec.Annotations.Name,
			ID:   resp2.ID,
		}
		inputs.stackID = resources[2].Spec.Annotations.Labels[types.StackLabel]
	})
	When("Uncreated resources", func() {
		BeforeEach(func() {
			badsearch := interfaces.ReconcileResource{}
			inputs.activeResource1, err = serviceInit.getActiveResource(badsearch)
		})
		It("mean active resources do not exist", func() {
			Expect(err).To(HaveOccurred())
			Expect(activeService{}).To(Equal(inputs.activeResource1))
		})
	})
	When("new resources are created", func() {
		It("they should not fail", func() {
			Expect(inputs.err1).ToNot(HaveOccurred())
			Expect(inputs.err2).ToNot(HaveOccurred())
		})
	})
	When("an unlabled resource is created", func() {
		SharedFailedResponseBehavior(&inputs)
	})
})

var _ = Describe("Initial Algorithm Support - Config", func() {
	var (
		cli        *fakes.FakeReconcilerClient
		configInit initializationConfig
		inputs     AlgorithmPluginInputs
		resources  []swarm.Config
		err        error
	)
	BeforeEach(func() {
		var id1, id2 string
		cli = fakes.NewFakeReconcilerClient()
		configInit = newInitializationSupportConfig(cli)
		// testing two configs one with a stacks label and
		// one without skipping zero-th element
		resources = fakes.GenerateConfigFixtures(3, "", "InitialConfig")
		id1, inputs.err1 = cli.CreateConfig(resources[1].Spec)
		id2, inputs.err2 = cli.CreateConfig(resources[2].Spec)
		inputs.algorithmInit = configInit
		inputs.search1.SnapshotResource = interfaces.SnapshotResource{
			Name: resources[1].Spec.Annotations.Name,
			ID:   id1,
		}
		inputs.search2.SnapshotResource = interfaces.SnapshotResource{
			Name: resources[2].Spec.Annotations.Name,
			ID:   id2,
		}
		inputs.stackID = resources[2].Spec.Annotations.Labels[types.StackLabel]
	})
	When("Uncreated resources", func() {
		BeforeEach(func() {
			badsearch := interfaces.ReconcileResource{}
			inputs.activeResource1, err = configInit.getActiveResource(badsearch)
		})
		It("mean active resources do not exist", func() {
			Expect(err).To(HaveOccurred())
			Expect(activeConfig{}).To(Equal(inputs.activeResource1))
		})
	})
	When("new resources are created", func() {
		It("they should not fail", func() {
			Expect(inputs.err1).ToNot(HaveOccurred())
			Expect(inputs.err2).ToNot(HaveOccurred())
		})
	})
	When("an unlabled resource is created", func() {
		SharedFailedResponseBehavior(&inputs)
	})
})

var _ = Describe("Initial Algorithm Support - Secret", func() {
	var (
		cli        *fakes.FakeReconcilerClient
		secretInit initializationSecret
		inputs     AlgorithmPluginInputs
		resources  []swarm.Secret
		err        error
	)
	BeforeEach(func() {
		var id1, id2 string
		cli = fakes.NewFakeReconcilerClient()
		secretInit = newInitializationSupportSecret(cli)
		// testing two secrets one with a stacks label and
		// one without skipping zero-th element
		resources = fakes.GenerateSecretFixtures(3, "", "InitialSecret")
		id1, inputs.err1 = cli.CreateSecret(resources[1].Spec)
		id2, inputs.err2 = cli.CreateSecret(resources[2].Spec)
		inputs.algorithmInit = secretInit
		inputs.search1.SnapshotResource = interfaces.SnapshotResource{
			Name: resources[1].Spec.Annotations.Name,
			ID:   id1,
		}
		inputs.search2.SnapshotResource = interfaces.SnapshotResource{
			Name: resources[2].Spec.Annotations.Name,
			ID:   id2,
		}
		inputs.stackID = resources[2].Spec.Annotations.Labels[types.StackLabel]
	})
	When("Uncreated resources", func() {
		BeforeEach(func() {
			badsearch := interfaces.ReconcileResource{}
			inputs.activeResource1, err = secretInit.getActiveResource(badsearch)
		})
		It("mean active resources do not exist", func() {
			Expect(err).To(HaveOccurred())
			Expect(activeSecret{}).To(Equal(inputs.activeResource1))
		})
	})
	When("new resources are created", func() {
		It("they should not fail", func() {
			Expect(inputs.err1).ToNot(HaveOccurred())
			Expect(inputs.err2).ToNot(HaveOccurred())
		})
	})
	When("an unlabled resource is created", func() {
		SharedFailedResponseBehavior(&inputs)
	})
})

var _ = Describe("Initial Algorithm Support - Network", func() {
	var (
		cli         *fakes.FakeReconcilerClient
		networkInit initializationNetwork
		inputs      AlgorithmPluginInputs
		resources   []dockerTypes.NetworkCreateRequest
		err         error
	)
	BeforeEach(func() {
		var id1, id2 string
		cli = fakes.NewFakeReconcilerClient()
		networkInit = newInitializationSupportNetwork(cli)
		// testing two networks one with a stacks label and
		// one without skipping zero-th element
		resources = fakes.GenerateNetworkFixtures(3, "", "InitialNetwork")
		id1, inputs.err1 = cli.CreateNetwork(resources[1])
		id2, inputs.err2 = cli.CreateNetwork(resources[2])
		inputs.algorithmInit = networkInit
		inputs.search1.SnapshotResource = interfaces.SnapshotResource{
			Name: resources[1].Name,
			ID:   id1,
		}
		inputs.search2.SnapshotResource = interfaces.SnapshotResource{
			Name: resources[2].Name,
			ID:   id2,
		}
		inputs.stackID = resources[2].NetworkCreate.Labels[types.StackLabel]
	})
	When("Uncreated resources", func() {
		BeforeEach(func() {
			badsearch := interfaces.ReconcileResource{}
			inputs.activeResource1, err = networkInit.getActiveResource(badsearch)
		})
		It("mean active resources do not exist", func() {
			Expect(err).To(HaveOccurred())
			Expect(activeNetwork{}).To(Equal(inputs.activeResource1))
		})
	})
	When("new resources are created", func() {
		It("they should not fail", func() {
			Expect(inputs.err1).ToNot(HaveOccurred())
			Expect(inputs.err2).ToNot(HaveOccurred())
		})
	})
	When("an unlabled resource is created", func() {
		SharedFailedResponseBehavior(&inputs)
	})
})
