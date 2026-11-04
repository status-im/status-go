package cliutils

import (
	"flag"
	"strings"
	"syscall"

	cli "github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
)

type GenericFlagSingleValue struct {
	*cli.GenericFlag
	set *flag.FlagSet
}

type GenericFlagMultiValue struct {
	*cli.GenericFlag
	set *flag.FlagSet
}

func NewGenericFlagSingleValue(fl *cli.GenericFlag) *GenericFlagSingleValue {
	return &GenericFlagSingleValue{GenericFlag: fl, set: nil}
}

func NewGenericFlagMultiValue(fl *cli.GenericFlag) *GenericFlagMultiValue {
	return &GenericFlagMultiValue{GenericFlag: fl, set: nil}
}

func (f *GenericFlagSingleValue) Apply(set *flag.FlagSet) error {
	f.set = set
	return f.GenericFlag.Apply(set)
}

func (f *GenericFlagMultiValue) Apply(set *flag.FlagSet) error {
	f.set = set
	return f.GenericFlag.Apply(set)
}

func isEnvVarSet(envVars []string) bool {
	for _, envVar := range envVars {
		if _, ok := syscall.Getenv(envVar); ok {
			// TODO: Can't use this for bools as
			// set means that it was true or false based on
			// Bool flag type, should work for other types
			return true
		}
	}

	return false
}

func (f *GenericFlagSingleValue) ApplyInputSourceValue(cCtx *cli.Context, isc altsrc.InputSourceContext) error {
	if f.set == nil || cCtx.IsSet(f.Name) || isEnvVarSet(f.EnvVars) {
		return nil
	}

	for _, name := range f.GenericFlag.Names() {
		value, err := isc.String(name)
		if err != nil {
			return err
		}
		if value == "" {
			continue
		}

		for _, n := range f.Names() {
			_ = f.set.Set(n, value)
		}
	}

	return nil
}

func (f *GenericFlagMultiValue) ApplyInputSourceValue(cCtx *cli.Context, isc altsrc.InputSourceContext) error {
	if f.set == nil || cCtx.IsSet(f.Name) || isEnvVarSet(f.EnvVars) {
		return nil
	}

	for _, name := range f.GenericFlag.Names() {
		value, err := isc.StringSlice(name)
		if err != nil {
			return err
		}
		if len(value) == 0 {
			continue
		}

		for _, n := range f.Names() {
			_ = f.set.Set(n, strings.Join(value, ","))
		}

	}

	return nil
}
