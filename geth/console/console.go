package console

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ethereum/go-ethereum/console"
	"github.com/mattn/go-colorable"
	"github.com/peterh/liner"
)

// HistoryFile is the file within the data directory to store input scrollback.
const HistoryFile = "history.log"

// DefaultPrompt is the default prompt line prefix to use for user input querying.
const DefaultPrompt = "> "

var (
	onlyWhitespace = regexp.MustCompile(`^\s*$`)
	exit           = regexp.MustCompile(`^\s*exit\s*;*\s*$`)
)

// Config is a collection of configuration optiosn of the Console.
type Config struct {
	DataDir      string               // Data directory to store the console history at
	Prompt       string               // Input prompt prefix string (defaults to DefaultPrompt)
	Prompter     console.UserPrompter // Input prompter to allow interactive user feedback (defaults to TerminalPrompter)
	Printer      io.Writer            // Output writer to serialize any display strings to (defaults to os.Stdout)
	Autocomplete []string             // A list of strings to autocomplete.
}

// Console is an interactive entrypoint to C bindings
// exported for the mobile clients.
type Console struct {
	prompt       string               // Input prompt prefix string
	prompter     console.UserPrompter // Input prompter to allow interactive user feedback
	histPath     string               // Absolute path to the console scrollback history
	history      []string             // Scroll history maintained by the console
	printer      io.Writer            // Output writer to serialize any display strings to
	autocomplete []string             // A list of strings to autocomplete.
}

// New returns a new Console instance.
// It sets some default values if config fields are empty.
func New(config Config) (*Console, error) {
	if config.Prompt == "" {
		config.Prompt = DefaultPrompt
	}
	if config.Prompter == nil {
		config.Prompter = console.Stdin
	}
	if config.Printer == nil {
		config.Printer = colorable.NewColorableStdout()
	}

	console := Console{
		prompt:       config.Prompt,
		prompter:     config.Prompter,
		printer:      config.Printer,
		histPath:     filepath.Join(config.DataDir, HistoryFile),
		autocomplete: config.Autocomplete,
	}
	if err := console.init(); err != nil {
		return nil, err
	}

	return &console, nil
}

func (c *Console) init() error {
	// Configure the console's input prompter for scrollback and tab completion.
	if c.prompter != nil {
		if content, err := ioutil.ReadFile(c.histPath); err != nil {
			c.prompter.SetHistory(nil)
		} else {
			c.history = strings.Split(string(content), "\n")
			c.prompter.SetHistory(c.history)
		}
		c.prompter.SetWordCompleter(c.autoCompleteInput)
	}

	return nil
}

func (c *Console) autoCompleteInput(line string, pos int) (string, []string, string) {
	// No completions can be provided for empty inputs.
	if len(line) == 0 || pos == 0 {
		return "", nil, ""
	}

	var words []string
	for _, word := range c.autocomplete {
		if strings.HasPrefix(word, line) {
			words = append(words, word)
		}
	}

	return "", words, line[pos:]
}

// Welcome prints a welcome message.
func (c *Console) Welcome() {
	fmt.Fprintf(c.printer, "Welcome to the Status Console!\n\n")
}

// Interactive starts an interactive user session, where input is prompted
// for an input.
// commandHandler is required, otherwise prompt is displayed before the command
// finishes and prints its output.
func (c *Console) Interactive(commandHandler <-chan string) <-chan string {
	// Channel to send the next prompt on and receive the input.
	scheduler := make(chan string)

	// Start a goroutine to listen for promt requests and send back inputs.
	go func() {
		for {
			// Read the next user input.
			line, err := c.prompter.PromptInput(<-scheduler)
			if err != nil {
				// In case of an error, either clear the prompt or fail.
				if err == liner.ErrPromptAborted { // ctrl-C
					scheduler <- ""
					continue
				}
				close(scheduler)
				return
			}
			// User input retrieved, send for interpretation and loop.
			scheduler <- line
		}
	}()

	// Monitor Ctrl-C too in case the input is empty and we need to bail.
	abort := make(chan os.Signal, 1)
	signal.Notify(abort, os.Interrupt)

	// Channel to send received commands.
	commands := make(chan string)
	// Prompt string. It can be changed by commandHandler.
	prompt := c.prompt

	// Start sending prompts to the user and reading back inputs
	go func() {
		defer close(commands)

		for {
			// Send the next prompt, triggering an input read and process the result
			scheduler <- prompt

			select {
			case <-abort:
				// User forcefully quite the console
				fmt.Fprintln(c.printer, "caught interrupt, exiting")
				return

			case line, ok := <-scheduler:
				// User input was returned by the prompter, handle special cases.
				if !ok || exit.MatchString(line) {
					return
				}
				if onlyWhitespace.MatchString(line) {
					continue
				}

				command := strings.TrimSpace(line)

				if len(c.history) == 0 || command != c.history[len(c.history)-1] {
					c.history = append(c.history, command)
					if c.prompter != nil {
						c.prompter.AppendHistory(command)
					}
				}

				commands <- command

				// Wait for the command to be finished.
				prompt = <-commandHandler
				// Empty prompt is not allowed. Change to default.
				if prompt == "" {
					prompt = c.prompt
				}
			}
		}
	}()

	return commands
}

// Stop cleans up the console and terminates the runtime envorinment.
func (c *Console) Stop() error {
	if err := ioutil.WriteFile(c.histPath, []byte(strings.Join(c.history, "\n")), 0600); err != nil {
		return err
	}
	if err := os.Chmod(c.histPath, 0600); err != nil { // Force 0600, even if it was different previously
		return err
	}

	return nil
}
