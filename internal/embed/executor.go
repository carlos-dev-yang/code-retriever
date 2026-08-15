package embed

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sort"
	"sync"
	"time"

	"cidx/internal/embedclient"
)

type Waiter func(context.Context, time.Duration) error

type ExecuteOptions struct {
	Limits         RequestLimits
	MaxConcurrency int
	AttemptTimeout time.Duration
	MaxRetries     int
	RetryWaits     []time.Duration
	Wait           Waiter
}

type Outcome struct {
	Group            RequestGroup
	Response         embedclient.EmbeddingResponse
	Vectors          [][]float32
	Attempts         int
	ResponseRejected bool
	Err              error
}

// HandlerError marks a local persistence/invariant failure. Callers must not
// continue a cancelled execution by recording new state after this error.
type HandlerError struct{ Err error }

func (e *HandlerError) Error() string { return "embedding success handler failed" }
func (e *HandlerError) Unwrap() error { return e.Err }

// Execute calls only the normal synchronous EmbeddingClient endpoint. A
// success handler runs serially as groups complete so durable stores can
// commit each validated group immediately. The returned outcomes are always
// ordinally ordered, regardless of completion order.
func Execute(ctx context.Context, client embedclient.EmbeddingClient, source embedclient.EmbeddingSourceSpec, role embedclient.InputRole, inputs []RequestInput, options ExecuteOptions, onSuccess func(context.Context, Outcome) error) ([]Outcome, error) {
	if client == nil {
		return nil, fmt.Errorf("embedding client is required")
	}
	if err := source.Validate(); err != nil {
		return nil, err
	}
	if role != embedclient.DocumentRole && role != embedclient.QueryRole {
		return nil, fmt.Errorf("embedding input role is invalid")
	}
	if options.MaxConcurrency <= 0 || options.AttemptTimeout <= 0 || options.MaxRetries < 0 || len(options.RetryWaits) < options.MaxRetries {
		return nil, fmt.Errorf("embedding execution policy is invalid")
	}
	groups, err := Group(inputs, options.Limits)
	if err != nil || len(groups) == 0 {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if options.Wait == nil {
		options.Wait = waitContext
	}
	execCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	jobs := make(chan RequestGroup)
	completed := make(chan Outcome, len(groups))
	workers := options.MaxConcurrency
	if workers > len(groups) {
		workers = len(groups)
	}
	var workerWG sync.WaitGroup
	for i := 0; i < workers; i++ {
		workerWG.Add(1)
		go func() {
			defer workerWG.Done()
			for group := range jobs {
				completed <- executeGroup(execCtx, client, source, role, group, options)
			}
		}()
	}
	next, received := 0, 0
	outcomes := make([]Outcome, 0, len(groups))
	stopAndDrain := func() []Outcome {
		cancel()
		close(jobs)
		workerWG.Wait()
		for received < next {
			outcomes = append(outcomes, <-completed)
			received++
		}
		return sortedOutcomes(outcomes)
	}
	for received < len(groups) {
		var send chan RequestGroup
		var job RequestGroup
		if next < len(groups) {
			send, job = jobs, groups[next]
		}
		select {
		case send <- job:
			next++
		case outcome := <-completed:
			received++
			outcomes = append(outcomes, outcome)
			if outcome.Err == nil && onSuccess != nil {
				if handlerErr := onSuccess(execCtx, outcome); handlerErr != nil {
					return stopAndDrain(), &HandlerError{Err: handlerErr}
				}
			}
		case <-ctx.Done():
			return stopAndDrain(), ctx.Err()
		}
	}
	close(jobs)
	workerWG.Wait()
	if err := ctx.Err(); err != nil {
		return sortedOutcomes(outcomes), err
	}
	return sortedOutcomes(outcomes), nil
}

func executeGroup(ctx context.Context, client embedclient.EmbeddingClient, source embedclient.EmbeddingSourceSpec, role embedclient.InputRole, group RequestGroup, options ExecuteOptions) Outcome {
	texts := make([]string, len(group.Inputs))
	for i, input := range group.Inputs {
		texts[i] = string(input.Bytes)
	}
	request := embedclient.EmbeddingRequest{Source: source, Role: role, Inputs: texts}
	outcome := Outcome{Group: group}
	for attempt := 0; ; attempt++ {
		outcome.Attempts++
		attemptCtx, cancel := context.WithTimeout(ctx, options.AttemptTimeout)
		response, err := client.Embed(attemptCtx, request)
		attemptErr := attemptCtx.Err()
		cancel()
		if attemptErr == context.DeadlineExceeded && ctx.Err() == nil {
			err = embedclient.ProviderError{Class: "timeout", Retryable: true, Cause: attemptErr}
		}
		if ctx.Err() != nil {
			outcome.Err = ctx.Err()
			return outcome
		}
		if err == nil {
			vectors, validationErr := embedclient.ValidateResponse(request, response)
			if validationErr != nil {
				outcome.Err, outcome.ResponseRejected = validationErr, true
				return outcome
			}
			outcome.Response, outcome.Vectors = response, vectors
			return outcome
		}
		if attempt == options.MaxRetries || !Transient(err) {
			outcome.Err = err
			return outcome
		}
		wait := options.RetryWaits[attempt]
		if retryAfter, ok := embedclient.RetryAfter(err); ok && retryAfter > wait {
			wait = retryAfter
		}
		if err := options.Wait(ctx, wait); err != nil {
			outcome.Err = err
			return outcome
		}
	}
}

func Transient(err error) bool {
	if embedclient.IsRetryable(err) {
		return true
	}
	var networkErr net.Error
	return errors.As(err, &networkErr) && (networkErr.Timeout() || networkErr.Temporary())
}

func waitContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func sortedOutcomes(outcomes []Outcome) []Outcome {
	sort.Slice(outcomes, func(i, j int) bool { return outcomes[i].Group.Ordinal < outcomes[j].Group.Ordinal })
	return outcomes
}
