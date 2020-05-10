package main

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
)

type Completer func(string) []string

type Command struct {
	name     string
	function interface{}
	complete Completer
}

func (c *Command) Complete(line string) []string {
	if c.complete != nil {
		return c.complete(line)
	}
	return []string{}
}

func (c *Command) AttachCompleter(completer Completer) {
	c.complete = completer
}

func (c *Command) Execute(args ...string) {
	if c.function == nil {
		return
	}

	v := reflect.ValueOf(c.function)
	if v.Kind() != reflect.Func {
		panic("Execute requires a function!")
	}

	t := v.Type()
	ni := t.NumIn()
	if len(args) < ni {
		panic("Arguments missing!")
	}

	var argTypes []reflect.Type
	for i := 0; i < ni; i++ {
		argTypes = append(argTypes, t.In(i))
	}
	//no := t.NumOut()

	var values []reflect.Value
	for i, arg := range args[:ni] {
		argValue, err := convertStringToType(argTypes[i], arg)
		if err != nil {
			return
		}
		values = append(values, argValue)
	}
	v.Call(values)
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
			return result, errors.New("Int value too big: " + strVal)
		}
		result.SetInt(val)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val, err := strconv.ParseUint(strVal, 0, 64)
		if err != nil {
			return result, err
		}
		if result.OverflowUint(val) {
			return result, errors.New("UInt value too big: " + strVal)
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
		return result, errors.New("Unsupported kind: " + t.Kind().String())
	}
	return result, nil
}

type commandError struct {
	err string
}

func (e *commandError) Error() string {
	return fmt.Sprintf("%s", e.err)
}
