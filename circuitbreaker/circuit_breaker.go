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
	Err       error
}

func (cr CommandResult) Result() []any {
	return cr.res
}

func (cr CommandResult) Error() error {
	return cr.err
}
func (cr CommandResult) Cancelled() bool {
	return cr.cancelled
}

func (cr CommandResult) FunctorCallStatuses() []FunctorCallStatus {
	return cr.functorCallStatuses
}

func (cr *CommandResult) addCallStatus(circuitName string, err error) {
	cr.functorCallStatuses = append(cr.functorCallStatuses, FunctorCallStatus{
		Name:      circuitName,
		Timestamp: time.Now(),
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

func (cmd *Command) Add(ftor *Functor) {
	cmd.functors = append(cmd.functors, ftor)
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
	exec        FallbackFunc
	circuitName string
}

func NewFunctor(exec FallbackFunc, circuitName string) *Functor {
	return &Functor{
		exec:        exec,
		circuitName: circuitName,
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

// Executes the command in its circuit if set.
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
		if cb.circuitNameHandler != nil {
			circuitName = cb.circuitNameHandler(circuitName)
		}

		// if last command, execute without circuit
		if i == len(cmd.functors)-1 {
			res, execErr := f.exec()
			err = execErr
			if err == nil {
				result.res = res
				result.err = nil
			}
			result.addCallStatus(circuitName, err)
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
				res, err := f.exec()
				// Write to result only if success
				if err == nil {
					result.res = res
					result.err = nil
				}
				result.addCallStatus(circuitName, err)

				// If the command has been cancelled, we don't count
				// the error towars breaking the circuit, and then we break
				if cmd.cancel {
					result = accumulateCommandError(result, circuitName, err)
					result.cancelled = true
					return nil
				}
				if err != nil {
					logutils.ZapLogger().Warn("hystrix error", zap.String("provider", circuitName), zap.Error(err))
				}
				return err
			}, nil)
		}
		if err == nil {
			break
		}

		result = accumulateCommandError(result, circuitName, err)

		// Lets abuse every provider with the same amount of MaxConcurrentRequests,
		// keep iterating even in case of ErrMaxConcurrency error
	}
	return result
}

func (c *CircuitBreaker) SetOverrideCircuitNameHandler(f func(string) string) {
	c.circuitNameHandler = f
}

// Expects a circuit to exist because a new circuit is always closed.
// Call CircuitExists to check if a circuit exists.
func IsCircuitOpen(circuitName string) bool {
	circuit, wasCreated, _ := hystrix.GetCircuit(circuitName)
	return !wasCreated && circuit.IsOpen()
}

func CircuitExists(circuitName string) bool {
	_, wasCreated, _ := hystrix.GetCircuit(circuitName)
	return !wasCreated
}
