package interfaces

import (
	"context"
	"fmt"
	"sync"
	"time"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"

	"github.com/docker/stacks/pkg/types"
)

// BackendAPIClientShim is an implementation of BackendClient that utilizes an
// in-memory FakeStackStore for Stacks CRUD, and an underlying Docker API
// Client for swarm operations. It is intended for use only as part of the
// standalone runtime of the stacks controller. Only one event subscriber is
// expected at any time.
type BackendAPIClientShim struct {
	dclient client.CommonAPIClient
	StackStore

	// The following constructs are used to generate events for stack
	// operations locally, and multiplex them into the daemon's event stream.
	stackEvents   chan events.Message
	subscribersMu sync.Mutex
	subscribers   map[chan interface{}]context.CancelFunc
}

// NewBackendAPIClientShim creates a new BackendAPIClientShim.
func NewBackendAPIClientShim(dclient client.CommonAPIClient) BackendClient {
	return &BackendAPIClientShim{
		dclient:     dclient,
		StackStore:  NewFakeStackStore(),
		stackEvents: make(chan events.Message),
	}
}

// GetServices lists services.
func (c *BackendAPIClientShim) GetServices(options dockerTypes.ServiceListOptions) ([]swarm.Service, error) {
	return c.dclient.ServiceList(context.Background(), options)
}

// GetService inspects a single service.
func (c *BackendAPIClientShim) GetService(idOrName string, insertDefaults bool) (swarm.Service, error) {
	svc, _, err := c.dclient.ServiceInspectWithRaw(context.Background(), idOrName, dockerTypes.ServiceInspectOptions{
		InsertDefaults: insertDefaults,
	})
	return svc, err
}

// CreateService creates a new service.
func (c *BackendAPIClientShim) CreateService(spec swarm.ServiceSpec, encodedRegistryAuth string, queryRegistry bool) (*dockerTypes.ServiceCreateResponse, error) {
	resp, err := c.dclient.ServiceCreate(context.Background(), spec, dockerTypes.ServiceCreateOptions{
		EncodedRegistryAuth: encodedRegistryAuth,
		QueryRegistry:       queryRegistry,
	})
	return &resp, err
}

// UpdateService updates a service.
func (c *BackendAPIClientShim) UpdateService(
	idOrName string,
	version uint64,
	spec swarm.ServiceSpec,
	options dockerTypes.ServiceUpdateOptions,
	queryRegistry bool,
) (*dockerTypes.ServiceUpdateResponse, error) {
	options.QueryRegistry = queryRegistry
	resp, err := c.dclient.ServiceUpdate(context.Background(), idOrName, swarm.Version{Index: version}, spec, options)
	return &resp, err
}

// RemoveService removes a service.
func (c *BackendAPIClientShim) RemoveService(idOrName string) error {
	return c.dclient.ServiceRemove(context.Background(), idOrName)
}

// GetTasks returns multiple tasks.
func (c *BackendAPIClientShim) GetTasks(options dockerTypes.TaskListOptions) ([]swarm.Task, error) {
	return c.dclient.TaskList(context.Background(), options)
}

// GetTask returns a task.
func (c *BackendAPIClientShim) GetTask(taskID string) (swarm.Task, error) {
	task, _, err := c.dclient.TaskInspectWithRaw(context.Background(), taskID)
	return task, err
}

// GetSecrets lists multiple secrets.
func (c *BackendAPIClientShim) GetSecrets(opts dockerTypes.SecretListOptions) ([]swarm.Secret, error) {
	return c.dclient.SecretList(context.Background(), opts)
}

// CreateSecret creates a secret.
func (c *BackendAPIClientShim) CreateSecret(s swarm.SecretSpec) (string, error) {
	resp, err := c.dclient.SecretCreate(context.Background(), s)
	return resp.ID, err
}

// RemoveSecret removes a secret.
func (c *BackendAPIClientShim) RemoveSecret(idOrName string) error {
	return c.dclient.SecretRemove(context.Background(), idOrName)
}

// GetSecret inspects a secret.
func (c *BackendAPIClientShim) GetSecret(id string) (swarm.Secret, error) {
	secret, _, err := c.dclient.SecretInspectWithRaw(context.Background(), id)
	return secret, err
}

// UpdateSecret updates a secret.
func (c *BackendAPIClientShim) UpdateSecret(idOrName string, version uint64, spec swarm.SecretSpec) error {
	return c.dclient.SecretUpdate(context.Background(), idOrName, swarm.Version{Index: version}, spec)
}

// GetConfigs lists multiple configs.
func (c *BackendAPIClientShim) GetConfigs(opts dockerTypes.ConfigListOptions) ([]swarm.Config, error) {
	return c.dclient.ConfigList(context.Background(), opts)
}

// CreateConfig creates a config.
func (c *BackendAPIClientShim) CreateConfig(s swarm.ConfigSpec) (string, error) {
	resp, err := c.dclient.ConfigCreate(context.Background(), s)
	return resp.ID, err
}

// RemoveConfig removes a config.
func (c *BackendAPIClientShim) RemoveConfig(id string) error {
	return c.dclient.ConfigRemove(context.Background(), id)
}

// GetConfig inspects a config.
func (c *BackendAPIClientShim) GetConfig(id string) (swarm.Config, error) {
	cfg, _, err := c.dclient.ConfigInspectWithRaw(context.Background(), id)
	return cfg, err
}

// UpdateConfig updates a config.
func (c *BackendAPIClientShim) UpdateConfig(idOrName string, version uint64, spec swarm.ConfigSpec) error {
	return c.dclient.ConfigUpdate(context.Background(), idOrName, swarm.Version{Index: version}, spec)
}

// SubscribeToEvents subscribes to the system event stream. The API Client's
// Events API has no way to distinguish between buffered and streamed events,
// thus even past are provided through the returned channel.
func (c *BackendAPIClientShim) SubscribeToEvents(since, until time.Time, ef filters.Args) ([]events.Message, chan interface{}) {
	ctx, cancel := context.WithCancel(context.Background())

	resChan := make(chan interface{})
	eventsChan, _ := c.dclient.Events(context.Background(), dockerTypes.EventsOptions{
		Filters: ef,
		Since:   fmt.Sprintf("%d", since.Unix()),
		Until:   fmt.Sprintf("%d", until.Unix()),
	})

	go func() {
		for {
			select {
			case event := <-c.stackEvents:
				resChan <- event
			case event := <-eventsChan:
				resChan <- event
			case <-ctx.Done():
				return
			}
		}
	}()

	c.subscribersMu.Lock()
	c.subscribers[resChan] = cancel
	c.subscribersMu.Unlock()

	return []events.Message{}, resChan
}

// UnsubscribeFromEvents unsubscribes from the event stream.
func (c *BackendAPIClientShim) UnsubscribeFromEvents(eventChan chan interface{}) {
	c.subscribersMu.Lock()
	defer c.subscribersMu.Unlock()

	if cancelFunc, ok := c.subscribers[eventChan]; ok {
		cancelFunc()
		delete(c.subscribers, eventChan)
	}
}

// AddStack creates a stack.
func (c *BackendAPIClientShim) AddStack(spec types.StackSpec) (string, error) {
	id, err := c.StackStore.AddStack(spec)
	if err != nil {
		return "", fmt.Errorf("unable to create stack: %s", err)
	}

	go func() {
		c.stackEvents <- events.Message{
			Type:   "stack",
			Action: "create",
			ID:     id,
		}
	}()

	return id, err
}

// UpdateStack updates a stack.
func (c *BackendAPIClientShim) UpdateStack(id string, spec types.StackSpec) error {
	err := c.StackStore.UpdateStack(id, spec)
	go func() {
		c.stackEvents <- events.Message{
			Type:   "stack",
			Action: "update",
			ID:     id,
		}
	}()

	return err
}

// DeleteStack deletes a stack.
func (c *BackendAPIClientShim) DeleteStack(id string) error {
	err := c.StackStore.DeleteStack(id)
	go func() {
		c.stackEvents <- events.Message{
			Type:   "stack",
			Action: "delete",
			ID:     id,
		}
	}()
	return err
}

// GetNetworks return a list of networks.
func (c *BackendAPIClientShim) GetNetworks(f filters.Args) ([]dockerTypes.NetworkResource, error) {
	return c.dclient.NetworkList(context.Background(), dockerTypes.NetworkListOptions{
		Filters: f,
	})
}

// GetNetwork inspects a network.
func (c *BackendAPIClientShim) GetNetwork(name string) (dockerTypes.NetworkResource, error) {
	return c.dclient.NetworkInspect(context.Background(), name, dockerTypes.NetworkInspectOptions{})
}

// GetNetworksByName is a great example of a bad interface design.
func (c *BackendAPIClientShim) GetNetworksByName(name string) ([]dockerTypes.NetworkResource, error) {
	f := filters.NewArgs()
	f.Add("name", name)
	return c.GetNetworks(f)
}

// CreateNetwork creates a new network.
func (c *BackendAPIClientShim) CreateNetwork(nc dockerTypes.NetworkCreateRequest) (string, error) {
	resp, err := c.dclient.NetworkCreate(context.Background(), nc.Name, nc.NetworkCreate)
	return resp.ID, err
}

// RemoveNetwork removes a network.
func (c *BackendAPIClientShim) RemoveNetwork(name string) error {
	return c.dclient.NetworkRemove(context.Background(), name)
}
