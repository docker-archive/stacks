package store

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"context"

	swarmapi "github.com/docker/swarmkit/api"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/docker/stacks/pkg/mocks"
)

var _ = Describe("Store functions", func() {
	Describe("InitExtension", func() {
		var (
			ctx                 context.Context
			ctrl                *gomock.Controller
			mockResourcesClient *mocks.MockResourcesClient
		)
		BeforeEach(func() {
			ctx = context.Background()
			ctrl = gomock.NewController(GinkgoT())
			mockResourcesClient = mocks.NewMockResourcesClient(ctrl)
		})
		AfterEach(func() {
			ctrl.Finish()
		})

		It("should ensure the extension exists", func() {
			mockResourcesClient.EXPECT().CreateExtension(
				ctx, &swarmapi.CreateExtensionRequest{
					Annotations: &swarmapi.Annotations{
						Name: StackResourceKind,
					},
					Description: StackResourcesDescription,
				},
			).Return(nil, nil)

			err := InitExtension(ctx, mockResourcesClient)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return no error if the extension already exists", func() {
			mockResourcesClient.EXPECT().CreateExtension(
				ctx, &swarmapi.CreateExtensionRequest{
					Annotations: &swarmapi.Annotations{
						Name: StackResourceKind,
					},
					Description: StackResourcesDescription,
				},
			).Return(nil, status.Errorf(codes.AlreadyExists, "already exists"))

			err := InitExtension(ctx, mockResourcesClient)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return an error if a different grpc error occurs", func() {
			mockResourcesClient.EXPECT().CreateExtension(
				ctx, &swarmapi.CreateExtensionRequest{
					Annotations: &swarmapi.Annotations{
						Name: StackResourceKind,
					},
					Description: StackResourcesDescription,
				},
			).Return(nil, status.Errorf(codes.Unknown, "busted"))

			err := InitExtension(ctx, mockResourcesClient)
			Expect(err).To(HaveOccurred())
		})

		It("should return an error if a non-grpc error occurs", func() {
			mockResourcesClient.EXPECT().CreateExtension(
				ctx, &swarmapi.CreateExtensionRequest{
					Annotations: &swarmapi.Annotations{
						Name: StackResourceKind,
					},
					Description: StackResourcesDescription,
				},
			).Return(nil, errors.New("busted"))

			err := InitExtension(ctx, mockResourcesClient)
			Expect(err).To(HaveOccurred())
		})
	})
})
