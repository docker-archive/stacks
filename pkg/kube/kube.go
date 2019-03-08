package kube

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/docker/compose-on-kubernetes/api/client/clientset"
	composev1beta2 "github.com/docker/compose-on-kubernetes/api/client/clientset/typed/compose/v1beta2"
	"github.com/docker/compose-on-kubernetes/api/compose/v1beta2"
	"github.com/docker/docker/errdefs"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apitypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"

	"github.com/docker/stacks/pkg/compose/loader"
	"github.com/docker/stacks/pkg/types"
)

// StacksBackend is a client.StackAPIClient implementation that uses a
// Kubernetes API client to access Kubernetes Stack CRD objects and surface
// them as Stacks.
// NOTE: This implementation should eventually be more coupled to the
// architecture of the compose-on-kubernetes project, so as to avoid all the
// complex type conversions present in this package.
// NOTE: StacksBackend attempts to be have no impact over the underlying
// compose-on-kubernetes CRD resources, which means that storing a stack in
// the Kubernetes backend loses data for all the fields not relevant to the
// Kubernetes backend, such as service.CgroupParents
type StacksBackend struct {
	composeClient composev1beta2.ComposeV1beta2Interface
	coreV1Client  corev1.CoreV1Interface
}

// NewStacksBackend creates a new StacksBackend.
func NewStacksBackend(c *rest.Config) (*StacksBackend, error) {
	kubeClient, err := kubernetes.NewForConfig(c)
	if err != nil {
		return nil, fmt.Errorf("unable to create Kubernetes clientset: %s", err)
	}

	stacksClient, err := clientset.NewForConfig(c)
	if err != nil {
		return nil, fmt.Errorf("unable to create Kubernetes clientset: %s", err)
	}

	return &StacksBackend{
		coreV1Client:  kubeClient.CoreV1(),
		composeClient: stacksClient.ComposeV1beta2(),
	}, nil
}

var errNotFound = errdefs.NotFound(errors.New("stack not found"))

// Multiple stacks may share the name across namespaces, but Stack IDs are
// unique and  represented as `kube_namespace_name`.
// NOTE: The `_` character is invalid as a name or namespace character in
// kubernetes, so it was chosen as a separator between the namespace and stack
// name. Ref:
// https://kubernetes.io/docs/concepts/overview/working-with-objects/names/

// getKubeStackID computes the ID for a Kubernetes Stack given its namespace and name.
func getKubeStackID(namespace, name string) string {
	return fmt.Sprintf("kube_%s_%s", namespace, name)
}

// parseKubeStackID extracts the namespace and stack name from a stack ID.
func parseKubeStackID(id string) (string, string, error) {
	parts := strings.Split(id, "_")
	if len(parts) != 3 || parts[0] != "kube" ||
		parts[1] == "" || parts[2] == "" {
		return "", "", fmt.Errorf("invalid ID format: %s", id)
	}

	namespace := parts[1]
	name := parts[2]

	return namespace, name, nil
}

// ParseComposeInput is a passthrough to the actual loader
// implementation.
func (c *StacksBackend) ParseComposeInput(_ context.Context, input types.ComposeInput) (*types.StackCreate, error) {
	return loader.ParseComposeInput(input)
}

// StackCreate creates a stack.
func (c *StacksBackend) StackCreate(_ context.Context, create types.StackCreate, _ types.StackCreateOptions) (types.StackCreateResponse, error) {
	kubeStack := FromStackSpec(create.Spec)

	_, err := c.composeClient.Stacks(kubeStack.ObjectMeta.Namespace).Create(kubeStack)
	if err != nil {
		return types.StackCreateResponse{}, err
	}

	return types.StackCreateResponse{
		ID: getKubeStackID(create.Spec.Collection, create.Spec.Metadata.Name),
	}, nil
}

// StackInspect inspects a stack.
func (c *StacksBackend) StackInspect(_ context.Context, id string) (types.Stack, error) {
	namespace, name, err := parseKubeStackID(id)
	if err != nil {
		// Any error in parsing the stack ID results in a "stack not
		// found" response, as the provided ID is not a valid ID for
		// the kube backend.
		return types.Stack{}, errNotFound
	}

	kubeStack, err := c.composeClient.Stacks(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return types.Stack{}, errNotFound
		}
		return types.Stack{}, fmt.Errorf("unable to get stack %s: %s", id, err)
	}

	// TODO: populate status
	return ConvertFromKubeStack(kubeStack)
}

// StackList lists all stacks.
func (c *StacksBackend) StackList(_ context.Context, _ types.StackListOptions) ([]types.Stack, error) {
	// Iterate over all namespaces. This assumes that the underlying
	// coreV1Client has `namespace list` permissions.
	nsResp, err := c.coreV1Client.Namespaces().List(metav1.ListOptions{})
	if err != nil {
		return []types.Stack{}, fmt.Errorf("unable to retrieve all namespaces: %s", err)
	}

	// Aggregate stacks across all namespaces
	// TODO: there's a race here in the event where a namespace was
	// just deleted.
	allStacks := []v1beta2.Stack{}
	for _, ns := range nsResp.Items {
		namespace := ns.Name
		resp, err := c.composeClient.Stacks(namespace).List(metav1.ListOptions{})
		if err != nil {
			return []types.Stack{}, fmt.Errorf("unable to list stacks in namespace %s: %s", namespace, err)
		}
		allStacks = append(allStacks, resp.Items...)
	}

	// TODO: populate statuses
	return ConvertFromKubeStacks(allStacks)
}

// StackUpdate updates a stack.
func (c *StacksBackend) StackUpdate(_ context.Context, id string, version types.Version, spec types.StackSpec, _ types.StackUpdateOptions) error {
	namespace, name, err := parseKubeStackID(id)
	if err != nil {
		// Any error in parsing the stack ID results in a "stack not
		// found" response, as the provided ID is not a valid ID for the
		// kube backend.
		return errNotFound
	}

	kubeStack := FromStackSpec(spec)
	kubeStack.ObjectMeta.ResourceVersion = fmt.Sprintf("%d", version.Index)

	patchBytes, err := json.Marshal(kubeStack)
	if err != nil {
		return fmt.Errorf("unable to marshal patch stack: %s", err)
	}

	_, err = c.composeClient.Stacks(namespace).Patch(name, apitypes.StrategicMergePatchType, patchBytes)
	if kerrors.IsNotFound(err) {
		return errNotFound
	}

	return err
}

// StackDelete deletes a stack.
func (c *StacksBackend) StackDelete(_ context.Context, id string) error {
	namespace, name, err := parseKubeStackID(id)
	if err != nil {
		// Any error in parsing the stack ID results in an "okay"
		// response, as delete is idempotent.
		return nil
	}

	return c.composeClient.Stacks(namespace).Delete(name, &metav1.DeleteOptions{})
}
