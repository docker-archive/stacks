package kube

import (
	"context"
	"testing"

	"github.com/docker/compose-on-kubernetes/api/compose/v1beta2"
	"github.com/docker/docker/errdefs"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apitypes "k8s.io/apimachinery/pkg/types"

	"github.com/docker/stacks/pkg/mocks"
	"github.com/docker/stacks/pkg/types"
)

func TestParseKubeStackID(t *testing.T) {
	require := require.New(t)

	namespace, name, err := parseKubeStackID("foobar")
	require.Error(err)
	require.Empty(namespace)
	require.Empty(name)

	namespace, name, err = parseKubeStackID("")
	require.Error(err)
	require.Empty(namespace)
	require.Empty(name)

	namespace, name, err = parseKubeStackID("kube__name")
	require.Error(err)
	require.Empty(namespace, "")
	require.Empty(name, "")

	namespace, name, err = parseKubeStackID("kube_namespace_")
	require.Error(err)
	require.Empty(namespace, "")
	require.Empty(name, "")

	namespace, name, err = parseKubeStackID("kube__")
	require.Error(err)
	require.Empty(namespace, "")
	require.Empty(name, "")

	namespace, name, err = parseKubeStackID("kube_namespace_name")
	require.NoError(err)
	require.Equal(namespace, "namespace")
	require.Equal(name, "name")
}

func TestKubeStacksBackendStackCreate(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := mocks.NewMockComposeV1beta2Interface(ctrl)
	b := StacksBackend{
		composeClient: s,
	}

	k := mocks.NewMockStackInterface(ctrl)
	s.EXPECT().Stacks("namespace1").Return(k)

	k.EXPECT().Create(gomock.Any()).Return(nil, nil)

	create := types.StackCreate{
		Spec: types.StackSpec{
			Metadata: types.Metadata{
				Name: "testname",
			},
			Collection: "namespace1",
		},
	}

	resp, err := b.StackCreate(context.TODO(), create, types.StackCreateOptions{})
	require.NoError(t, err)
	require.Equal(t, resp.ID, "kube_namespace1_testname")
}

func TestKubeStacksBackendStackInspect(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	s := mocks.NewMockComposeV1beta2Interface(ctrl)
	b := StacksBackend{
		composeClient: s,
	}

	k := mocks.NewMockStackInterface(ctrl)
	s.EXPECT().Stacks("namespace1").Return(k)

	k.EXPECT().Get("testname", gomock.Any()).Return(&v1beta2.Stack{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "testname",
			Namespace:       "namespace1",
			ResourceVersion: "42",
			Annotations: map[string]string{
				"key": "value",
			},
		},
		Spec: &v1beta2.StackSpec{
			Services: []v1beta2.ServiceConfig{
				{
					Image: "service1",
				},
			},
		},
	}, nil)

	stack, err := b.StackInspect(context.TODO(), "kube_namespace1_testname")
	require.NoError(err)
	require.Equal(stack.Version.Index, uint64(42))
	require.Equal(stack.Spec.Collection, "namespace1")
	require.Equal(stack.Spec.Metadata.Name, "testname")
	require.Equal(stack.Spec.Metadata.Labels["key"], "value")
	require.Len(stack.Spec.Services, 1)
	require.Equal(stack.Spec.Services[0].Image, "service1")
}

func TestKubeStacksBackendStackInspectNotFoundInvalid(t *testing.T) {
	b := StacksBackend{}

	stack, err := b.StackInspect(context.TODO(), "failid")
	require.Error(t, err)
	require.True(t, errdefs.IsNotFound(err))
	require.Empty(t, stack)
}

func TestKubeStacksBackendStackInspectNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := mocks.NewMockComposeV1beta2Interface(ctrl)
	b := StacksBackend{
		composeClient: s,
	}

	k := mocks.NewMockStackInterface(ctrl)
	s.EXPECT().Stacks("namespace1").Return(k)

	k.EXPECT().Get("testname", gomock.Any()).Return(&v1beta2.Stack{}, kerrors.NewNotFound(v1beta2.GroupResource("stack"), "testname"))

	stack, err := b.StackInspect(context.TODO(), "kube_namespace1_testname")
	require.Error(t, err)
	require.True(t, errdefs.IsNotFound(err))
	require.Empty(t, stack)
}

func TestKubeStacksBackendStackDelete(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := mocks.NewMockComposeV1beta2Interface(ctrl)
	b := StacksBackend{
		composeClient: s,
	}

	k := mocks.NewMockStackInterface(ctrl)
	s.EXPECT().Stacks("namespace1").Return(k)

	k.EXPECT().Delete("testname", gomock.Any()).Return(nil)

	err := b.StackDelete(context.TODO(), "kube_namespace1_testname")
	require.NoError(t, err)
}

func TestKubeStacksBackendStackList(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	s := mocks.NewMockComposeV1beta2Interface(ctrl)
	l := mocks.NewMockCoreV1Interface(ctrl)
	b := StacksBackend{
		composeClient: s,
		coreV1Client:  l,
	}

	j := mocks.NewMockStackInterface(ctrl)
	k := mocks.NewMockStackInterface(ctrl)
	h := mocks.NewMockStackInterface(ctrl)
	s.EXPECT().Stacks("namespace1").Return(k)
	s.EXPECT().Stacks("namespace2").Return(j)
	s.EXPECT().Stacks("namespace3").Return(h)

	ns := mocks.NewMockNamespaceInterface(ctrl)
	l.EXPECT().Namespaces().Return(ns)
	ns.EXPECT().List(gomock.Any()).Return(&corev1.NamespaceList{
		Items: []corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "namespace1",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "namespace2",
				},
			},
		},
	}, nil)

	k.EXPECT().List(gomock.Any()).Return(&v1beta2.StackList{
		Items: []v1beta2.Stack{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "testname",
					Namespace:       "namespace1",
					ResourceVersion: "46",
				},
				Spec: &v1beta2.StackSpec{
					Services: []v1beta2.ServiceConfig{
						{
							Image: "service1",
						},
					},
				},
			},
		},
	}, nil)

	j.EXPECT().List(gomock.Any()).Return(&v1beta2.StackList{
		Items: []v1beta2.Stack{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "testname",
					Namespace:       "namespace2",
					ResourceVersion: "44",
				},
				Spec: &v1beta2.StackSpec{
					Services: []v1beta2.ServiceConfig{
						{
							Image: "service2",
						},
					},
				},
			},
		},
	}, nil)

	h.EXPECT().List(gomock.Any()).Return(&v1beta2.StackList{
		Items: []v1beta2.Stack{},
	}, nil)

	stacks, err := b.StackList(context.TODO(), types.StackListOptions{})
	require.NoError(err)
	require.Len(stacks, 2)

	found := map[string]string{
		"namespace1": "service1",
		"namespace2": "service2",
	}

	for _, stack := range stacks {
		require.Equal(stack.Spec.Metadata.Name, "testname")
		targetImage, ok := found[stack.Spec.Collection]
		require.True(ok)
		require.Equal(stack.Spec.Services[0].Image, targetImage)
		delete(found, stack.Spec.Collection)
	}

	require.Len(found, 0)
}

func TestKubeStacksBackendStackUpdate(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := mocks.NewMockComposeV1beta2Interface(ctrl)
	b := StacksBackend{
		composeClient: s,
	}

	k := mocks.NewMockStackInterface(ctrl)
	s.EXPECT().Stacks("namespace1").Return(k)
	k.EXPECT().Patch("testname", apitypes.StrategicMergePatchType, gomock.Any()).Return(&v1beta2.Stack{}, nil)

	err := b.StackUpdate(context.TODO(), "kube_namespace1_testname", types.Version{}, types.StackSpec{}, types.StackUpdateOptions{})
	require.NoError(t, err)
}

func TestKubeStacksBackendStackNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := mocks.NewMockComposeV1beta2Interface(ctrl)
	b := StacksBackend{
		composeClient: s,
	}

	k := mocks.NewMockStackInterface(ctrl)
	s.EXPECT().Stacks("namespace1").Return(k)

	k.EXPECT().Patch("testname", apitypes.StrategicMergePatchType, gomock.Any()).Return(&v1beta2.Stack{}, kerrors.NewNotFound(v1beta2.GroupResource("stack"), "testname"))

	err := b.StackUpdate(context.TODO(), "kube_namespace1_testname", types.Version{}, types.StackSpec{}, types.StackUpdateOptions{})
	require.Error(t, err)
	require.True(t, errdefs.IsNotFound(err))
}

func TestKubeStacksBackendStackUpdateNotFoundInvalid(t *testing.T) {
	b := StacksBackend{}

	err := b.StackUpdate(context.TODO(), "failid", types.Version{}, types.StackSpec{}, types.StackUpdateOptions{})
	require.Error(t, err)
	require.True(t, errdefs.IsNotFound(err))
}
