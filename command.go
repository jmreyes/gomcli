package gomcli

import (
	"errors"
	"reflect"
	"strconv"
)

var ErrCmdMissingArgs = errors.New("missing arguments")
var ErrCmdArgIntOverflow = errors.New("Int value too big")
var ErrCmdArgUIntOverflow = errors.New("UInt value too big")
var ErrCmdArgUnsupportedKind = errors.New("Unsupported kind")

type Completer func(string) []string

type ErrHandler func(Command, error) error

type Command struct {
	Name       string
	Function   interface{}
	Completer  Completer
	ErrHandler ErrHandler
}

func (c *Command) complete(line string) []string {
	if c.Completer != nil {
		return c.Completer(line)
	}
	return []string{}
}

func (c *Command) handleErr(err error) error {
	if c.ErrHandler == nil {
		return err
	}
	retErr := c.ErrHandler(*c, err)
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
		err := c.handleErr(ErrCmdMissingArgs)
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
			err = c.handleErr(err)
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
			return result, ErrCmdArgIntOverflow
		}
		result.SetInt(val)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val, err := strconv.ParseUint(strVal, 0, 64)
		if err != nil {
			return result, err
		}
		if result.OverflowUint(val) {
			return result, ErrCmdArgUIntOverflow
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
