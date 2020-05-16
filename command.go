package gomcli

import (
	"errors"
	"reflect"
	"strconv"
)

// ErrCmdMissingArgs is passed to ErrHandler when the number of arguments
// provided via CLI for a Command is less than the number of arguments for
// its defined Function.
var ErrCmdMissingArgs = errors.New("Missing arguments")

// ErrCmdInvalidArgs is passed to ErrHandler when the arguments provided via
// CLI for a Command cannot be converted to the argument types for its defined
// Function.
var ErrCmdInvalidArgs = errors.New("Invalid arguments")

// ErrCmdArgOverflow is passed to ErrHandler when the value provided via CLI
// overflows the type of the corresponding argument for its defined Function.
var ErrCmdArgOverflow = errors.New("Value too big")

// ErrCmdArgUnsupportedKind is passed to ErrHandler when the Kind of a Function's
// argument is not supported.
var ErrCmdArgUnsupportedKind = errors.New("Unsupported Kind")

// Completer takes a string and returns a list of completion candidates. It can be
// set for a given Command to indicate gomcli how to complete subcommands.
type Completer func(string) []string

// ErrHandler takes a Command, an input string and a given error when parsing
// said Command, and returns an error to be propagated to GomCLI.Start, if needed.
// Otherwise, this is the point where the errors from CLI input for a Command
// are to be gracefully handled.
type ErrHandler func(*Command, []string, error) error

// Command represents a function that can be executed via the CLI. Name defines the
// string that needs to be provided via the CLI to execute the Function. ErrHandler
// allows to handle errors when converting the input to arguments for the Function.
// Completer allows to provide completions for subcommands.
type Command struct {
	Name       string
	Function   interface{}
	ErrHandler ErrHandler
	Completer  Completer
}

func (c *Command) complete(line string) []string {
	if c.Completer != nil {
		return c.Completer(line)
	}
	return []string{}
}

func (c *Command) handleErr(err error, args []string) error {
	if c.ErrHandler == nil {
		return err
	}
	retErr := c.ErrHandler(c, args, err)
	if retErr != nil {
		return retErr
	}
	return nil
}

func (c *Command) execute(args ...string) error {
	if c.Function == nil {
		panic("Execute requires a function!")
	}

	v := reflect.ValueOf(c.Function)
	if v.Kind() != reflect.Func {
		panic("Execute requires a function!")
	}

	t := v.Type()
	ni := t.NumIn()

	argsLen := len(args)
	if argsLen < ni {
		err := c.handleErr(ErrCmdMissingArgs, args)
		if err != nil {
			return err
		}
	}

	var argTypes []reflect.Type
	for i := 0; i < ni; i++ {
		argTypes = append(argTypes, t.In(i))
	}

	var values []reflect.Value
	for i, arg := range args[:ni] {
		argValue, err := convertStringToType(argTypes[i], arg)
		if err != nil {
			err = c.handleErr(err, args)
			if err != nil {
				return err
			}
		}
		values = append(values, argValue)
	}

	v.Call(values)

	return nil
}

// Borrowed from https://stackoverflow.com/q/39891689
func convertStringToType(t reflect.Type, strVal string) (reflect.Value, error) {
	result := reflect.Indirect(reflect.New(t))
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val, err := strconv.ParseInt(strVal, 0, 64)
		if err != nil {
			return result, err
		}
		if result.OverflowInt(val) {
			return result, ErrCmdArgOverflow
		}
		result.SetInt(val)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val, err := strconv.ParseUint(strVal, 0, 64)
		if err != nil {
			return result, err
		}
		if result.OverflowUint(val) {
			return result, ErrCmdArgOverflow
		}
		result.SetUint(val)
	case reflect.Float32:
		val, err := strconv.ParseFloat(strVal, 32)
		if err != nil {
			return result, err
		}
		result.SetFloat(val)
	case reflect.Float64:
		val, err := strconv.ParseFloat(strVal, 64)
		if err != nil {
			return result, err
		}
		result.SetFloat(val)
	case reflect.String:
		result.SetString(strVal)
	case reflect.Bool:
		val, err := strconv.ParseBool(strVal)
		if err != nil {
			return result, err
		}
		result.SetBool(val)
	default:
		return result, ErrCmdArgUnsupportedKind
	}
	return result, nil
}
