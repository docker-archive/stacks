package reconciler

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/docker/docker/api/types/swarm"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"

	// strictly speaking, we don't have to use errdefs here -- no error
	// checking actually occurs. However, to future-proof the tests, in case
	// something happens later, we should return errdefs errors
	"github.com/docker/docker/errdefs"

	"github.com/docker/stacks/pkg/mocks"
)

var _ = Describe("reconciler.Manager", func() {
	var (
		m *Manager

		ctrl       *gomock.Controller
		mockClient *mocks.MockBackendClient
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockClient = mocks.NewMockBackendClient(ctrl)

		m = New(mockClient)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("checkLeadership", func() {
		It("should return false if there is no node ID set", func() {
			mockClient.EXPECT().GetNode("").Return(
				swarm.Node{},
				errdefs.NotFound(errors.New("not found")),
			)

			Expect(m.checkLeadership()).To(BeFalse())
		})

		When("the nodeID field is set", func() {
			BeforeEach(func() {
				m.nodeID = "foo"
			})

			It("should return false if the node is not a manager", func() {
				mockClient.EXPECT().GetNode("foo").Return(
					// no ManagerStatus field, no error
					swarm.Node{ID: "foo"}, nil,
				)

				Expect(m.checkLeadership()).To(BeFalse())
			})

			It("should return false if the node is not the leader", func() {
				mockClient.EXPECT().GetNode("foo").Return(
					swarm.Node{
						ID: "foo", ManagerStatus: &swarm.ManagerStatus{
							Leader: false,
						},
					}, nil,
				)

				Expect(m.checkLeadership()).To(BeFalse())
			})

			It("should return true if the node is the leader", func() {
				mockClient.EXPECT().GetNode("foo").Return(
					swarm.Node{
						ID: "foo", ManagerStatus: &swarm.ManagerStatus{
							Leader: true,
						},
					}, nil,
				)

				Expect(m.checkLeadership()).To(BeTrue())
			})
		})
	})
})
