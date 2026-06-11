package circuitbreaker

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/afex/hystrix-go/hystrix"
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/logutils"
)

type FallbackFunc func() ([]any, error)

type CommandResult struct {
	res                 []any
	err                 error
	functorCallStatuses []FunctorCallStatus
	cancelled           bool
}

type FunctorCallStatus struct {
	Name      string
	Timestamp time.Time
	StartTime time.Time
	Err       error
}

func (cr *CommandResult) Result() []any {
	return cr.res
}

func (cr *CommandResult) Error() error {
	return cr.err
}

func (cr *CommandResult) Cancelled() bool {
	return cr.cancelled
}

func (cr *CommandResult) FunctorCallStatuses() []FunctorCallStatus {
	return cr.functorCallStatuses
}

func (cr *CommandResult) addCallStatus(providerName string, err error, startTime time.Time) {
	cr.functorCallStatuses = append(cr.functorCallStatuses, FunctorCallStatus{
		Name:      providerName,
		Timestamp: time.Now(),
		StartTime: startTime,
		Err:       err,
	})
}

type Command struct {
	ctx                       context.Context
	functors                  []*Functor
	cancel                    atomic.Bool
	nonFailureErrorClassifier func(error) bool
}

func NewCommand(ctx context.Context, functors []*Functor) *Command {
	return &Command{
		ctx:      ctx,
		functors: functors,
	}
}

func (cmd *Command) Add(functor *Functor) {
	cmd.functors = append(cmd.functors, functor)
}

func (cmd *Command) IsEmpty() bool {
	return len(cmd.functors) == 0
}

func (cmd *Command) Cancel() {
	cmd.cancel.Store(true)
}

func (cmd *Command) SetNonFailureErrorClassifier(classifier func(error) bool) {
	cmd.nonFailureErrorClassifier = classifier
}

func (cmd *Command) shouldIgnoreFailure(err error) bool {
	return err != nil && cmd.nonFailureErrorClassifier != nil && cmd.nonFailureErrorClassifier(err)
}

type Config struct {
	Timeout                int
	MaxConcurrentRequests  int
	RequestVolumeThreshold int
	SleepWindow            int
	ErrorPercentThreshold  int
}

type CircuitBreaker struct {
	config             Config
	circuitNameHandler func(string) string
}

func NewCircuitBreaker(config Config) *CircuitBreaker {
	return &CircuitBreaker{
		config: config,
	}
}

type Functor struct {
	exec         FallbackFunc
	circuitName  string
	providerName string
}

// NewFunctor creates a new Functor with the provided FallbackFunc, circuitName and providerName.
// The circuitName is the name of the circuit to be used by the Functor. If the circuitName is empty,
// or there is only one Functor in the Command, the command will be executed without a circuit.
func NewFunctor(exec FallbackFunc, circuitName, providerName string) *Functor {
	return &Functor{
		exec:         exec,
		circuitName:  circuitName,
		providerName: providerName,
	}
}

func accumulateCommandError(result CommandResult, circuitName string, err error) CommandResult {
	// Accumulate errors
	if result.err != nil {
		result.err = fmt.Errorf("%w, %s.error: %w", result.err, circuitName, err)
	} else {
		result.err = fmt.Errorf("%s.error: %w", circuitName, err)
	}
	return result
}

// sharedResult guards CommandResult against the hystrix goroutine that may
// outlive DoC on timeout or cancellation.
type sharedResult struct {
	mu     sync.Mutex
	result CommandResult
}

func (s *sharedResult) recordCall(providerName string, res []any, err error, startTime time.Time) FunctorCallStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err == nil {
		s.result.res = res
		s.result.err = nil
	}
	s.result.addCallStatus(providerName, err, startTime)
	return s.result.functorCallStatuses[len(s.result.functorCallStatuses)-1]
}

func (s *sharedResult) accumulateError(name string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.result = accumulateCommandError(s.result, name, err)
}

func (s *sharedResult) markCancelled() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.result.cancelled = true
}

func (s *sharedResult) snapshot() CommandResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := s.result
	cp.res = append([]any(nil), s.result.res...)
	cp.functorCallStatuses = append([]FunctorCallStatus(nil), s.result.functorCallStatuses...)
	return cp
}

// Execute the command in its circuit if set.
// If the command's circuit is not configured, the circuit of the CircuitBreaker is used.
// This is a blocking function.
func (cb *CircuitBreaker) Execute(cmd *Command) CommandResult {
	if cmd == nil || cmd.IsEmpty() {
		return CommandResult{err: fmt.Errorf("command is nil or empty")}
	}

	var shared sharedResult
	ctx := cmd.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	for i, f := range cmd.functors {
		if cmd.cancel.Load() {
			shared.markCancelled()
			break
		}

		var err error
		var skippedErr error
		circuitName := f.circuitName
		providerName := f.providerName
		if cb.circuitNameHandler != nil {
			circuitName = cb.circuitNameHandler(circuitName)
		}

		// Record start time before execution
		startTime := time.Now()

		// if last command, execute without circuit
		if i == len(cmd.functors)-1 || circuitName == "" {
			res, execErr := f.exec()
			err = execErr
			status := shared.recordCall(f.providerName, res, err, startTime)

			// Log error if present
			if err != nil {
				logutils.ZapLogger().Warn("direct execution error",
					zap.String("provider", f.providerName),
					zap.Error(err),
					zap.Duration("duration", time.Since(status.StartTime)))
			}
		} else {
			if hystrix.GetCircuitSettings()[circuitName] == nil {
				hystrix.ConfigureCommand(circuitName, hystrix.CommandConfig{
					Timeout:                cb.config.Timeout,
					MaxConcurrentRequests:  cb.config.MaxConcurrentRequests,
					RequestVolumeThreshold: cb.config.RequestVolumeThreshold,
					SleepWindow:            cb.config.SleepWindow,
					ErrorPercentThreshold:  cb.config.ErrorPercentThreshold,
				})
			}

			err = hystrix.DoC(ctx, circuitName, func(ctx context.Context) error {
				// We need to capture the execution time inside the hystrix function
				execStartTime := time.Now()
				res, err := f.exec()
				status := shared.recordCall(f.providerName, res, err, execStartTime)

				// If the command has been cancelled, we don't count
				// the error towards breaking the circuit, and then we break
				if cmd.cancel.Load() {
					shared.accumulateError(circuitName, err)
					shared.markCancelled()
					return nil
				}
				if cmd.shouldIgnoreFailure(err) {
					skippedErr = err
					return nil
				}
				if err != nil {
					logutils.ZapLogger().Warn("hystrix error",
						zap.String("provider", f.providerName),
						zap.Error(err),
						zap.Duration("duration", time.Since(status.StartTime)))
				}
				return err
			}, nil)
		}
		if err == nil && skippedErr != nil {
			err = skippedErr
		}
		if err == nil {
			break
		}

		shared.accumulateError(providerName, err)
		// Let's abuse every provider with the same amount of MaxConcurrentRequests,
		// keep iterating even in case of ErrMaxConcurrency error
	}
	return shared.snapshot()
}

func (cb *CircuitBreaker) SetOverrideCircuitNameHandler(f func(string) string) {
	cb.circuitNameHandler = f
}

// IsCircuitOpen Expects a circuit to exist because a new circuit is always closed.
func IsCircuitOpen(circuitName string) bool {
	circuit, wasCreated, _ := hystrix.GetCircuit(circuitName)
	return !wasCreated && circuit.IsOpen()
}

// CircuitExists checks if a circuit exists.
func CircuitExists(circuitName string) bool {
	_, wasCreated, _ := hystrix.GetCircuit(circuitName)
	return !wasCreated
}
