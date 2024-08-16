package circuitbreaker

import (
	"context"
	"fmt"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/ethereum/go-ethereum/log"
)

type FallbackFunc func() ([]any, error)

type CommandResult struct {
	res []any
	err error
}

func (cr CommandResult) Result() []any {
	return cr.res
}

func (cr CommandResult) Error() error {
	return cr.err
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

	for _, f := range cmd.functors {
		if cmd.cancel {
			break
		}

		circuitName := f.circuitName
		if cb.circuitNameHandler != nil {
			circuitName = cb.circuitNameHandler(circuitName)
		}

		if hystrix.GetCircuitSettings()[circuitName] == nil {
			hystrix.ConfigureCommand(circuitName, hystrix.CommandConfig{
				Timeout:                cb.config.Timeout,
				MaxConcurrentRequests:  cb.config.MaxConcurrentRequests,
				RequestVolumeThreshold: cb.config.RequestVolumeThreshold,
				SleepWindow:            cb.config.SleepWindow,
				ErrorPercentThreshold:  cb.config.ErrorPercentThreshold,
			})
		}

		err := hystrix.DoC(ctx, circuitName, func(ctx context.Context) error {
			res, err := f.exec()
			// Write to result only if success
			if err == nil {
				result = CommandResult{res: res}
			} else {
				log.Error("histrix-breaker error", "error", err)
			}
			return err
		}, nil)

		if err == nil {
			break
		}

		// Accumulate errors
		if result.err != nil {
			result.err = fmt.Errorf("%w, %s.error: %w", result.err, f.circuitName, err)
		} else {
			result.err = fmt.Errorf("%s.error: %w", f.circuitName, err)
		}
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
