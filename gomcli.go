package gomcli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/anmitsu/go-shlex"
	"github.com/peterh/liner"
)

// ErrCliPromptAborted is returned from Start or StartWithInput when the
// user presses Ctrl-C, if CtrlCAborts was set to true in the Conf struct.
var ErrCliPromptAborted = errors.New("prompt aborted")

// ErrCliCannotParseLine is returned from Start or StartWithInput if the
// input provided could not be be parsed to form command and arguments.
var ErrCliCannotParseLine = errors.New("cannot parse line")

// ErrCliCommandNotFound is passed to the notFoundHandler function if the input
// provided does not match any known command.
var ErrCliCommandNotFound = errors.New("command not found")

// NotFoundHandler is a function that indicates gomcli how to handle input
// that does not match any known Command. If not set, default action is to ignore
// it. An error can be returned, that will be propagated so that it is returned
// by Start.
type NotFoundHandler func(string) error

// GomCLI represents the state of the command-line interface, and is the main
// object to interact with within your program.
type GomCLI struct {
	lr              *liner.State
	prompt          string
	histfile        string
	commands        map[string]Command
	notFoundHandler NotFoundHandler
}

// New initializes a new *GomCLI with sane defaults. Further configuration is
// to be performed via the setters. The terminal is set to raw mode by Liner's
// action, therefore to restore the terminal to its previous state,
// GomCLI.Close() needs to be called.
func New() *GomCLI {
	c := &GomCLI{}
	c.prompt = "gomcli > "
	c.commands = make(map[string]Command)

	c.lr = liner.NewLiner()
	c.lr.SetTabCompletionStyle(liner.TabPrints)

	return c
}

// SetPrompt sets the prompt for the CLI. Note that due to Liner's multi-platform
// nature, colored prompts are not supported.
func (c *GomCLI) SetPrompt(prompt string) {
	c.prompt = prompt
}

// SetCtrlCAborts sets whether Start will return an ErrPromptAborted when Ctrl-C
// is pressed. The default is false (will not return when Ctrl-C is pressed).
func (c *GomCLI) SetCtrlCAborts(aborts bool) {
	c.lr.SetCtrlCAborts(aborts)
}

// SetNotFoundHandler sets the function that will be called when the provided input
// does not match any known Command.
func (c *GomCLI) SetNotFoundHandler(function NotFoundHandler) {
	c.notFoundHandler = function
}

// SetHistoryFile sets the path for the command history file. If not set, no history
// file will be used. The history file has a fixed limit of 1000 entries.
func (c *GomCLI) SetHistoryFile(path string) {
	c.histfile = path
	c.setupHistory()
}

// AddCommand adds a single Command to the CLI.
func (c *GomCLI) AddCommand(cmd Command) {
	c.commands[cmd.Name] = cmd
}

// SetCommands replaces the current CLI set of Commands by a new slice.
func (c *GomCLI) SetCommands(cmds []Command) {
	c.commands = make(map[string]Command)
	for _, cmd := range cmds {
		c.commands[cmd.Name] = cmd
	}
}

// RemoveCommand removes a specific Command from the CLI by name.
func (c *GomCLI) RemoveCommand(name string) {
	delete(c.commands, name)
}

// Commands retrieves the map with the current list of Commands for the CLI.
func (c *GomCLI) Commands() map[string]Command {
	return c.commands
}

func (c *GomCLI) setupHistory() {
	if c.histfile == "" {
		return
	}

	f, err := os.Open(c.histfile)
	if err != nil {
		return
	}
	c.lr.ReadHistory(f)
	f.Close()
}

func (c *GomCLI) writeHistory() error {
	if c.histfile == "" {
		return nil
	}

	dirName := filepath.Dir(c.histfile)
	if _, err := os.Stat(dirName); err != nil {
		err := os.MkdirAll(dirName, os.ModePerm)
		if err != nil {
			return err
		}
	}

	f, err := os.Create(c.histfile)
	if err != nil {
		return err
	}
	c.lr.WriteHistory(f)
	f.Close()

	return nil
}

func (c *GomCLI) complete(line string, pos int) (head string, comp []string, tail string) {
	tokens, _ := shlex.Split(line[:pos], false)
	tail = line[pos:]
	for i := len(tokens); i > 0; i-- {
		chunk := strings.Join(tokens[:i], " ")
		if cmd, err := c.getCommand(chunk); err == nil {
			if i == len(tokens) {
				return line, cmd.complete(""), tail
			}
			search := tokens[i]
			return cmd.Name + " ", cmd.complete(search), tail
		}
	}
	return head, c.rawCommandCompleter(line), tail
}

func (c *GomCLI) contextualComplete() []string {
	keys := make([]string, 0, len(c.commands))
	for k := range c.commands {
		keys = append(keys, k)
	}
	return keys
}

func (c *GomCLI) rawCommandCompleter(line string) (res []string) {
	for _, cmd := range c.contextualComplete() {
		if strings.HasPrefix(cmd, line) {
			res = append(res, cmd)
		}
	}
	return
}

func (c *GomCLI) getCommand(name string) (*Command, error) {
	if cmd, ok := c.commands[name]; ok {
		return &cmd, nil
	}
	return nil, ErrCliCommandNotFound
}

func (c *GomCLI) process() error {
	userInput, err := c.lr.Prompt(c.prompt)
	if err != nil {
		return err
	}

	c.lr.AppendHistory(userInput)

	return c.processInput(userInput)
}

func (c *GomCLI) processInput(input string) error {
	lines, err := splitInlineCommands(input)
	if err != nil {
		return err
	}

	for _, line := range lines {
		err := c.processLine(line)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *GomCLI) processLine(line string) error {
	tokens, err := shlex.Split(line, true)
	if err != nil {
		return ErrCliCannotParseLine
	}

	if len(tokens) == 0 {
		return nil
	}

	for i := len(tokens); i > 0; i-- {
		chunk := strings.Join(tokens[:i], " ")
		cmd, err := c.getCommand(chunk)
		if err != nil {
			continue
		}

		if len(tokens) > 1 {
			return cmd.execute(tokens[i:]...)
		}
		return cmd.execute()
	}

	if c.notFoundHandler != nil {
		err = c.notFoundHandler(tokens[0])
	}
	return err
}

func splitInlineCommands(userInput string) ([]string, error) {
	parsed, err := shlex.Split(userInput, false)
	if err != nil {
		return nil, ErrCliCannotParseLine
	}

	lines := []string{}
	command := []string{}

	for i := 0; i < len(parsed); i++ {
		element := parsed[i]
		if len(element) > 1 && element[len(element)-2:] == "\\;" {
			command = append(command, element)
		} else if len(element) > 1 && element[len(element)-2:] == ";;" {
			return nil, ErrCliCannotParseLine
		} else if element[len(element)-1:] == ";" {
			if element[:len(element)-1] != "" {
				command = append(command, element[:len(element)-1])
			}
			lines = append(lines, strings.Join(command, " "))
			command = []string{}
		} else {
			command = append(command, element)
		}
	}

	if len(command) > 0 {
		lines = append(lines, strings.Join(command, " "))
	}

	return lines, nil
}

// StartWithInput starts the CLI by providing initial input that will
// be split into lines and, if applicable, into commands.
func (c *GomCLI) StartWithInput(input string) error {
	if err := c.processInput(input); err != nil {
		return err
	}

	return c.Start()
}

// Start starts the CLI, iteratively displaying the prompt and handling
// user input until Close is called or an error is returned during user input
// processing.
func (c *GomCLI) Start() error {
	defer c.Close()

	for {
		if err := c.process(); err != nil {
			switch err {
			case liner.ErrPromptAborted:
				return ErrCliPromptAborted
			default:
				return err
			}
		}
	}
}

// Close stops the CLI processing, updating the history file if applicable and
// resetting the terminal into its previous mode.
func (c *GomCLI) Close() {
	c.writeHistory()
	c.lr.Close()
}
