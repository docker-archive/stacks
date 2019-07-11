package router

import (
	"context"
	"fmt"
	"sync"

	"github.com/docker/docker/errdefs"
	"github.com/sirupsen/logrus"

	"github.com/docker/stacks/pkg/client"
	"github.com/docker/stacks/pkg/types"
)

// StacksRouter is a router for the Stacks API, responsible for routing
// requests to specific orchestrator backends. It implements the
// StackAPIClient interface.
type StacksRouter struct {
	backends map[types.OrchestratorChoice]client.StackAPIClient
}

// stackPair is used internally by the router to share implementations
// between Inspect and Update.
type stackPair struct {
	stack       types.Stack
	fromBackend types.OrchestratorChoice
}

// NewStacksRouter creates a new StacksRouter
func NewStacksRouter() *StacksRouter {
	return &StacksRouter{
		backends: make(map[types.OrchestratorChoice]client.StackAPIClient),
	}
}

// RegisterBackend registers a new orchestration backend for the
// StacksRouter. If a backend already exists for the specified
// orchestrator type, it will be overridden.
func (s *StacksRouter) RegisterBackend(orch types.OrchestratorChoice, stackClient client.StackAPIClient) {
	s.backends[orch] = stackClient
}

func (s *StacksRouter) getStack(ctx context.Context, id string) (stackPair, error) {
	stackChan := make(chan stackPair)
	errChan := make(chan error)
	spreadWG := sync.WaitGroup{}
	collectWG := sync.WaitGroup{}
	for backendType, backend := range s.backends {
		spreadWG.Add(1)
		go func(backendType types.OrchestratorChoice, backend client.StackAPIClient) {
			defer spreadWG.Done()
			stack, err := backend.StackInspect(ctx, id)
			if err != nil {
				if !errdefs.IsNotFound(err) {
					errChan <- fmt.Errorf("unable to look for stacks in backend %s: %s", backendType, err)
				}
				return
			}
			stackChan <- stackPair{
				stack:       stack,
				fromBackend: backendType,
			}
		}(backendType, backend)
	}

	var stackPairs []stackPair
	var errs []error

	// The following two goroutines collect results and errors from the
	// respective channels. A separate WaitGroup is used to avoid a race
	// between the `append` assignments and further reading of the errs and
	// stackPairs arrays by the main goroutine.
	collectWG.Add(2)
	go func() {
		defer collectWG.Done()
		for err := range errChan {
			errs = append(errs, err)
		}
	}()

	go func() {
		defer collectWG.Done()
		for stackPair := range stackChan {
			stackPairs = append(stackPairs, stackPair)
		}
	}()

	// Wait until the inspects are finished on the underlying backends.
	spreadWG.Wait()
	close(errChan)
	close(stackChan)

	// Wait until the collection goroutines are done collecting errors and
	// results from their channels.
	collectWG.Wait()

	if len(errs) > 0 {
		errMsg := fmt.Sprintf("encountered 1 backend error")
		if len(errs) > 1 {
			errMsg = fmt.Sprintf("encountered %d backend errors", len(errs))
		}
		for i, err := range errs {
			errMsg = fmt.Sprintf("%s (error %d): %s", errMsg, i, err)
		}

		return stackPair{}, fmt.Errorf(errMsg)
	}

	switch len(stackPairs) {
	case 0:
		return stackPair{}, errdefs.NotFound(fmt.Errorf("stack not found"))
	case 1:
		return stackPairs[0], nil
	default:
		return stackPair{}, fmt.Errorf("multiple instances of the requested stack detected across backends")
	}
}

// StackCreate creates a new stack.
func (s *StacksRouter) StackCreate(ctx context.Context, spec types.StackSpec, options types.StackCreateOptions) (types.StackCreateResponse, error) {
	var orchestrator types.OrchestratorChoice = types.OrchestratorSwarm
	backend, ok := s.backends[orchestrator]
	if !ok {
		return types.StackCreateResponse{}, fmt.Errorf("invalid orchestrator choice %s", orchestrator)
	}

	return backend.StackCreate(ctx, spec, options)
}

// StackInspect attempts to inspect a stack across all backends in parallel,
// and returns the first response.
func (s *StacksRouter) StackInspect(ctx context.Context, id string) (types.Stack, error) {
	stackPair, err := s.getStack(ctx, id)
	return stackPair.stack, err
}

// StackList lists all stacks across all backends.
func (s *StacksRouter) StackList(ctx context.Context, options types.StackListOptions) ([]types.Stack, error) {
	allStacks := []types.Stack{}
	for backendType, backend := range s.backends {
		stacks, err := backend.StackList(ctx, options)
		if err != nil {
			return []types.Stack{}, fmt.Errorf("unable to list stacks from backend %s: %s", backendType, err)
		}
		allStacks = append(allStacks, stacks...)
	}

	// TODO: sort/order stacks by some criteria

	return allStacks, nil
}

// StackUpdate identifies which backend an existing stack is located at, and
// calls the update operation of that backend.
func (s *StacksRouter) StackUpdate(ctx context.Context, id string, version types.Version, spec types.StackSpec, options types.StackUpdateOptions) error {
	stackPair, err := s.getStack(ctx, id)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return err
		}
		return fmt.Errorf("unable to look for stack: %s", err)
	}

	backend, ok := s.backends[stackPair.fromBackend]
	if !ok {
		return fmt.Errorf("internal error: no such backend %s", stackPair.fromBackend)
	}

	return backend.StackUpdate(ctx, id, version, spec, options)
}

// StackDelete deletes a stack from all backends. StackDelete should be
// idempotent so any errors need to be reported back.
func (s *StacksRouter) StackDelete(ctx context.Context, id string) error {
	for backendType, backend := range s.backends {
		logrus.Debugf("Deleting stack %s from backend %s", id, backendType)
		err := backend.StackDelete(ctx, id)
		if err != nil {
			return fmt.Errorf("unable to delete stack from backend %s: %s", backendType, err)
		}
	}

	return nil
}
