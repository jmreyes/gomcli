package main

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
	name       string
	function   interface{}
	complete   Completer
	errHandler ErrHandler
}

func (c *Command) Complete(line string) []string {
	if c.complete != nil {
		//fmt.Printf("\nSearching: \"%v\"\n", line)
		return c.complete(line)
	}
	return []string{}
}

func (c *Command) AttachCompleter(completer Completer) {
	c.complete = completer
}

func (c *Command) handleErr(err error) error {
	if c.errHandler == nil {
		return err
	}
	retErr := c.errHandler(*c, err)
	if retErr != nil {
		return retErr
	}
	return nil
}

func (c *Command) Execute(args ...string) error {
	if c.function == nil {
		panic("Execute requires a function!")
	}

	v := reflect.ValueOf(c.function)
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

// Borrowed from https://stackoverflow.com/questions/39891689/how-to-convert-a-string-value-to-the-correct-reflect-kind-in-go
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
