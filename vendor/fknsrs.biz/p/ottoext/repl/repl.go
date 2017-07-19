// Package repl implements an event loop aware REPL (read-eval-print loop)
// for otto.
package repl // import "fknsrs.biz/p/ottoext/repl"

import (
	"fmt"
	"io"
	"strings"

	"fknsrs.biz/p/ottoext/loop"
	"fknsrs.biz/p/ottoext/loop/looptask"
	"github.com/robertkrimen/otto"
	"github.com/robertkrimen/otto/parser"
	"gopkg.in/readline.v1"
)

// Run creates a REPL with the default prompt and no prelude.
func Run(l *loop.Loop) error {
	return RunWithPromptAndPrelude(l, "", "")
}

// RunWithPrompt runs a REPL with the given prompt and no prelude.
func RunWithPrompt(l *loop.Loop, prompt string) error {
	return RunWithPromptAndPrelude(l, prompt, "")
}

// RunWithPrelude runs a REPL with the default prompt and the given prelude.
func RunWithPrelude(l *loop.Loop, prelude string) error {
	return RunWithPromptAndPrelude(l, "", prelude)
}

// RunWithPromptAndPrelude runs a REPL with the given prompt and prelude.
func RunWithPromptAndPrelude(l *loop.Loop, prompt, prelude string) error {
	if prompt == "" {
		prompt = ">"
	}

	prompt = strings.Trim(prompt, " ")
	prompt += " "

	rl, err := readline.New(prompt)
	if err != nil {
		return err
	}

	l.VM().Set("console", map[string]interface{}{
		"log": func(c otto.FunctionCall) otto.Value {
			s := make([]string, len(c.ArgumentList))
			for i := 0; i < len(c.ArgumentList); i++ {
				s[i] = c.Argument(i).String()
			}

			rl.Stdout().Write([]byte(strings.Join(s, " ") + "\n"))
			rl.Refresh()

			return otto.UndefinedValue()
		},
		"warn": func(c otto.FunctionCall) otto.Value {
			s := make([]string, len(c.ArgumentList))
			for i := 0; i < len(c.ArgumentList); i++ {
				s[i] = c.Argument(i).String()
			}

			rl.Stderr().Write([]byte(strings.Join(s, " ") + "\n"))
			rl.Refresh()

			return otto.UndefinedValue()
		},
	})

	if prelude != "" {
		if _, err := io.Copy(rl.Stderr(), strings.NewReader(prelude+"\n")); err != nil {
			return err
		}

		rl.Refresh()
	}

	var d []string

	for {
		ll, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				if d != nil {
					d = nil

					rl.SetPrompt(prompt)
					rl.Refresh()

					continue
				}

				break
			}

			return err
		}

		if len(d) == 0 && ll == "" {
			continue
		}

		d = append(d, ll)
		s := strings.Join(d, "\n")

		if _, err := parser.ParseFile(nil, "repl", s, 0); err != nil {
			rl.SetPrompt(strings.Repeat(" ", len(prompt)))
		} else {
			rl.SetPrompt(prompt)

			d = nil

			t := looptask.NewEvalTask(s)
			// don't report errors to the loop - this lets us handle them and
			// resume normal operation
			t.SoftError = true
			l.Add(t)
			l.Ready(t)

			v, err := <-t.Value, <-t.Error
			if err != nil {
				if oerr, ok := err.(*otto.Error); ok {
					io.Copy(rl.Stdout(), strings.NewReader(oerr.String()))
				} else {
					io.Copy(rl.Stdout(), strings.NewReader(err.Error()))
				}
			} else {
				f, err := format(v, 80, 2, 5)
				if err != nil {
					panic(err)
				}

				rl.Stdout().Write([]byte("\r" + f + "\n"))
			}
		}

		rl.Refresh()
	}

	return rl.Close()
}

func inspect(v otto.Value, width, indent int) string {
	switch {
	case v.IsBoolean(), v.IsNull(), v.IsNumber(), v.IsString(), v.IsUndefined(), v.IsNaN():
		return fmt.Sprintf("%s%q", strings.Repeat("  ", indent), v.String())
	default:
		return ""
	}
}
