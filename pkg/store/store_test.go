package store

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/swarm"
	swarmapi "github.com/docker/swarmkit/api"
	gogotypes "github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"

	"github.com/docker/stacks/pkg/interfaces"
	"github.com/docker/stacks/pkg/mocks"
	"github.com/docker/stacks/pkg/types"
)

var _ = Describe("StackStore", func() {
	// NOTE(dperny): You're probably asking why we test the StackStore object
	// instead of the stacks functions directly. The answer is just that I
	// refactored the code into functions (for reuse) and did not want to have
	// to rewrite all of the tests.
	It("should conform to the types.StackStore interface", func() {
		// This doesn't actually contain any useful assertions, it'll just fail
		// at build time. However, we have to include at least one use of the
		// variable s or the build will also fail.
		var s interfaces.StackStore
		// create a new StackStore from scratch, instead of through the
		// constructor, because we don't have a client
		s = &StackStore{}
		Expect(s).ToNot(BeNil())
	})

	Describe("CRUD Methods", func() {
		var (
			s *StackStore

			mockController *gomock.Controller
			mockClient     *mocks.MockResourcesClient

			stackSpec     *types.StackSpec
			stackResource *swarmapi.Resource

			timeProto *gogotypes.Timestamp
			timeObj   time.Time
		)

		BeforeEach(func() {
			mockController = gomock.NewController(GinkgoT())
			mockClient = mocks.NewMockResourcesClient(mockController)

			s = New(mockClient)

			// these are essentially the same stacks from marshal_test.go
			stackSpec = &types.StackSpec{
				Annotations: swarm.Annotations{
					Name: "someName",
					Labels: map[string]string{
						"key": "value",
					},
				},
				Services: []swarm.ServiceSpec{
					{
						Annotations: swarm.Annotations{Name: "bar"},
					},
				},
			}

			timeProto = gogotypes.TimestampNow()
			var err error
			timeObj, err = gogotypes.TimestampFromProto(timeProto)
			Expect(err).ToNot(HaveOccurred())

			stackSpecAny, err := MarshalStackSpec(stackSpec)
			Expect(err).ToNot(HaveOccurred())
			// we're allowed to use MarshalStackSpec in this as part of the test
			// code and not the code-under-test, because its correctness is
			// checked as part of marshal_test.go
			stackResource = &swarmapi.Resource{
				ID: "someID",
				Annotations: swarmapi.Annotations{
					Name:   stackSpec.Annotations.Name,
					Labels: stackSpec.Annotations.Labels,
				},
				Meta: swarmapi.Meta{
					CreatedAt: timeProto,
					UpdatedAt: timeProto,
					Version: swarmapi.Version{
						Index: 1,
					},
				},
				Payload: stackSpecAny,
			}
		})

		Specify("AddStack", func() {
			mockClient.EXPECT().CreateResource(
				context.TODO(),
				&swarmapi.CreateResourceRequest{
					Annotations: &stackResource.Annotations,
					Kind:        StackResourceKind,
					Payload:     stackResource.Payload,
				},
			).Return(
				&swarmapi.CreateResourceResponse{
					Resource: stackResource,
				},
				nil,
			)

			id, err := s.AddStack(*stackSpec)
			Expect(err).ToNot(HaveOccurred())
			Expect(id).To(Equal(stackResource.ID))
		})

		Specify("UpdateStack", func() {
			mockClient.EXPECT().GetResource(
				context.TODO(),
				&swarmapi.GetResourceRequest{
					ResourceID: stackResource.ID,
				},
			).Return(
				&swarmapi.GetResourceResponse{
					Resource: stackResource,
				},
				nil,
			)

			// create new stack objects that are matching our expectations.
			// additionally, include the information from the stackResource
			// object
			updatedStack := types.Stack{
				ID: stackResource.ID,
				Meta: swarm.Meta{
					Version: swarm.Version{
						Index: stackResource.Meta.Version.Index,
					},
					CreatedAt: timeObj,
					UpdatedAt: timeObj,
				},
				Spec: types.StackSpec{
					Annotations: swarm.Annotations{
						Name: "someName",
						Labels: map[string]string{
							"key": "value",
						},
					},
					Services: []swarm.ServiceSpec{
						{
							Annotations: swarm.Annotations{Name: "baz"},
						},
					},
				},
			}

			// marshal the specs just like the code under test would
			newAny, err := MarshalStackSpec(&updatedStack.Spec)
			Expect(err).ToNot(HaveOccurred())
			newResource := &swarmapi.Resource{
				ID:          stackResource.ID,
				Annotations: stackResource.Annotations,
				Meta: swarmapi.Meta{
					CreatedAt: timeProto,
					UpdatedAt: timeProto,
					Version: swarmapi.Version{
						Index: 2,
					},
				},
				Payload: newAny,
			}

			// now create an expectation that we'll update the resource like
			// this
			mockClient.EXPECT().UpdateResource(
				context.TODO(),
				&swarmapi.UpdateResourceRequest{
					ResourceID:      stackResource.ID,
					ResourceVersion: &stackResource.Meta.Version,
					Annotations:     &stackResource.Annotations,
					Payload:         newAny,
				},
			).Return(
				&swarmapi.UpdateResourceResponse{Resource: newResource}, nil,
			)

			err = s.UpdateStack(
				stackResource.ID,
				updatedStack.Spec,
				stackResource.Meta.Version.Index,
			)

			Expect(err).ToNot(HaveOccurred())
		})

		Specify("DeleteStack", func() {
			mockClient.EXPECT().RemoveResource(
				context.TODO(),
				&swarmapi.RemoveResourceRequest{ResourceID: stackResource.ID},
			).Return(
				&swarmapi.RemoveResourceResponse{}, nil,
			)

			err := s.DeleteStack(stackResource.ID)
			Expect(err).ToNot(HaveOccurred())
		})

		Specify("GetStack", func() {
			expectedStackWithFields := types.Stack{
				ID: stackResource.ID,
				Meta: swarm.Meta{
					Version: swarm.Version{
						Index: stackResource.Meta.Version.Index,
					},
					CreatedAt: timeObj,
					UpdatedAt: timeObj,
				},
				Spec: *stackSpec,
			}
			mockClient.EXPECT().GetResource(
				context.TODO(),
				&swarmapi.GetResourceRequest{
					ResourceID: stackResource.ID,
				},
			).Return(
				&swarmapi.GetResourceResponse{
					Resource: stackResource,
				}, nil,
			)
			resStack, err := s.GetStack(stackResource.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(resStack).To(Equal(expectedStackWithFields))
		})

		Describe("Listing", func() {
			var (
				numListedResources = 10

				allStackResources []*swarmapi.Resource
				// these are slices of interface{} so we can pass them straight
				// to ConsistOf
				allStacks []interface{}
			)
			BeforeEach(func() {
				for i := 0; i < numListedResources; i++ {
					stSpec := types.StackSpec{
						Annotations: swarm.Annotations{
							Name: fmt.Sprintf("stack_%v", i),
							Labels: map[string]string{
								"key": "value",
							},
						},
						Services: []swarm.ServiceSpec{
							{
								Annotations: swarm.Annotations{
									Name: fmt.Sprintf("svc_%v", i),
								},
							},
						},
					}

					// marshal the stacks
					any, err := MarshalStackSpec(&stSpec)
					Expect(err).ToNot(HaveOccurred())

					res := &swarmapi.Resource{
						ID: fmt.Sprintf("id_%v", i),
						Meta: swarmapi.Meta{
							Version: swarmapi.Version{
								Index: uint64(i),
							},
							CreatedAt: timeProto,
							UpdatedAt: timeProto,
						},
						Annotations: swarmapi.Annotations{
							Name:   stSpec.Annotations.Name,
							Labels: stSpec.Annotations.Labels,
						},
						Kind:    StackResourceKind,
						Payload: any,
					}
					allStackResources = append(allStackResources, res)

					// now, unmarshal the stacks so that we can put them in the
					// list of stacks with all the fields filled in
					unst, err := ConstructStack(res)
					Expect(err).ToNot(HaveOccurred())
					allStacks = append(allStacks, unst)
				}

				mockClient.EXPECT().ListResources(
					context.TODO(),
					&swarmapi.ListResourcesRequest{
						Filters: &swarmapi.ListResourcesRequest_Filters{
							Kind: StackResourceKind,
						},
					},
				).Return(
					&swarmapi.ListResourcesResponse{
						Resources: allStackResources,
					}, nil,
				)
			})

			Specify("ListStacks", func() {
				stacks, err := s.ListStacks()
				Expect(err).ToNot(HaveOccurred())
				Expect(stacks).To(ConsistOf(allStacks...))
			})
		})

		AfterEach(func() {
			mockController.Finish()
		})
	})
})
