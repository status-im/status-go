package circuitbreaker

import (
	"context"
	"fmt"
	"time"

	"github.com/afex/hystrix-go/hystrix"
	"go.uber.org/zap"

	"github.com/status-im/status-go/logutils"
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
	ctx      context.Context
	functors []*Functor
	cancel   bool
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
	cmd.cancel = true
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

// Execute the command in its circuit if set.
// If the command's circuit is not configured, the circuit of the CircuitBreaker is used.
// This is a blocking function.
func (cb *CircuitBreaker) Execute(cmd *Command) CommandResult {
	if cmd == nil || cmd.IsEmpty() {
		return CommandResult{err: fmt.Errorf("command is nil or empty")}
	}

	var result CommandResult
	ctx := cmd.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	for i, f := range cmd.functors {
		if cmd.cancel {
			result.cancelled = true
			break
		}

		var err error
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
			if err == nil {
				result.res = res
				result.err = nil
			}
			result.addCallStatus(f.providerName, err, startTime)

			// Log error if present
			if err != nil {
				// Get the status that was just added
				status := result.functorCallStatuses[len(result.functorCallStatuses)-1]
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
				// Write to result only if success
				if err == nil {
					result.res = res
					result.err = nil
				}
				result.addCallStatus(f.providerName, err, execStartTime)

				// If the command has been cancelled, we don't count
				// the error towards breaking the circuit, and then we break
				if cmd.cancel {
					result = accumulateCommandError(result, circuitName, err)
					result.cancelled = true
					return nil
				}
				if err != nil {
					// Get the status that was just added
					status := result.functorCallStatuses[len(result.functorCallStatuses)-1]
					logutils.ZapLogger().Warn("hystrix error",
						zap.String("provider", f.providerName),
						zap.Error(err),
						zap.Duration("duration", time.Since(status.StartTime)))
				}
				return err
			}, nil)
		}
		if err == nil {
			break
		}

		result = accumulateCommandError(result, providerName, err)
		// Let's abuse every provider with the same amount of MaxConcurrentRequests,
		// keep iterating even in case of ErrMaxConcurrency error
	}
	return result
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
