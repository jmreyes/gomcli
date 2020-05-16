# gomcli

*gomcli* is a simple Go library to help developers build interactive command-line interfaces in the style of [Metasploit Framework](https://www.metasploit.com/)'s *msfconsole*.

It is heavily inspired by [riposte](https://github.com/fwkz/riposte), a similar library for Python (in fact, some chunks of the code are a direct conversion to Go). However, the scope is a bit narrower, since things like output formatting are intentionally left out of the library's functionality.

![render1589661810858](https://user-images.githubusercontent.com/1638459/82129899-b5584080-97c6-11ea-96f8-3e77edbebc06.gif)

It uses the [Liner](https://github.com/peterh/liner) package under the hood. Therefore, many features already come for free, such as tab autocompletion, reverse search history, clear screen...

## Usage and example

Instantiate a `GomCLI`, set a custom prompt and you are good to go.

```go
import "github.com/jmreyes/gomcli"

func main() {
	cli = gomcli.New()
	cli.SetPrompt("myprompt$ ")
}

```

Add `Command`s to this instance to inform the CLI of the available options for autocompletion and execution. A `Command` is defined as follows:

```go
type Command struct {
	Name       string
	Function   interface{}
	ErrHandler ErrHandler
	Completer  Completer
}
```

- `Name`: The input that will trigger the execution of the `Function`.
- `Function`: The function that will be called.
- `ErrHandler`: Function to allow you to decide what happens when there are arguments missing, or invalid arguments are provided (e.g. provided `string` cannot be converted to the `int` argument the `Function` is expecting).
- `Completer`: Function that returns the completions for this `Command`, to allow for subcommands. The subcommands will be additional `Command`s, with apropriate `Name`.

Check out the [godoc](https://godoc.org/github.com/jmreyes/gomcli) for advanced configuration.

The following example tries to illustrate the basics, providing the functionality shown in the gif above.

```go
package main

import (
	"math"
	"strings"

	"github.com/jmreyes/gomcli"
)

var cli *gomcli.GomCLI

var currentMode = "basic"

func sum(x int, y int) {
	gomcli.Printf("[+] Sum is %v\n\n", x+y)
}

func mode() {
	gomcli.Printf("[i] Current mode is %v\n\n", currentMode)
}

func modeCompleter(text string) (res []string) {
	for _, subcommand := range []string{"basic", "advanced"} {
		if strings.HasPrefix(subcommand, text) {
			res = append(res, subcommand)
		}
	}
	return
}

func modeAdvanced() {
	currentMode = "advanced"
	cli.SetPrompt("gomcli [advanced] > ")

	gomcli.Printf("[+] Advanced mode set!\n\n")

	cli.AddCommand(gomcli.Command{
		Name: "power",
		Function: func(x float64, y float64) {
			gomcli.Printf("[+] Power is %v\n\n", math.Pow(x, y))
		},
		ErrHandler: errorHandler,
	})
}

func errorHandler(c *gomcli.Command, s []string, err error) error {
    // Check out the godoc for a full list of errors!
    if err == gomcli.ErrCmdMissingArgs {
		gomcli.Printf("[-] Arguments missing!\n\n")
	} else {
		gomcli.Printf("[-] Error! Did you really use valid input?\n\n")
	}
	return nil
}

func main() {
	cli = gomcli.New()

	cli.SetPrompt("gomcli > ")
	cli.SetNotFoundHandler(func(cmd string) error {
		gomcli.Printf("[-] Command %v not found!\n\n", cmd)
		return nil
	})

	cli.SetCommands([]gomcli.Command{
		{
			Name:       "sum",
			Function:   sum,
			ErrHandler: errorHandler,
		},
		{
			Name:      "mode",
			Function:  mode,
			Completer: modeCompleter,
		},
		{
			Name:     "mode advanced",
			Function: modeAdvanced,
		},
	})

	cli.Start()
}
```

## Dependencies

* [Liner](https://github.com/peterh/liner)
* [go-shlex](github.com/anmitsu/go-shlex)