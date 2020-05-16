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

// NotFoundHandler indicates gomcli how to handle unknown commands. If not set,
// unknown commands will be ignored. An error can be returned, that will be
// propagated so that the Start function returns it.
type NotFoundHandler func(string) error

type GomCLI struct {
	lr              *liner.State
	prompt          string
	histfile        string
	commands        map[string]Command
	notFoundHandler NotFoundHandler
}

func New() *GomCLI {
	c := &GomCLI{}
	c.prompt = "gomcli > "
	c.commands = make(map[string]Command)

	c.lr = liner.NewLiner()
	c.lr.SetTabCompletionStyle(liner.TabPrints)

	return c
}

func (c *GomCLI) SetPrompt(prompt string) {
	c.prompt = prompt
}

func (c *GomCLI) SetCtrlCAborts(aborts bool) {
	c.lr.SetCtrlCAborts(aborts)
}

func (c *GomCLI) SetNotFoundHandler(function NotFoundHandler) {
	c.notFoundHandler = function
}

func (c *GomCLI) SetHistoryFile(path string) {
	c.histfile = path
	c.setupHistory()
}

func (c *GomCLI) AddCommand(cmd Command) {
	c.commands[cmd.Name] = cmd
}

func (c *GomCLI) SetCommands(cmds []Command) {
	for _, cmd := range cmds {
		c.commands[cmd.Name] = cmd
	}
}

func (c *GomCLI) RemoveCommand(name string) {
	delete(c.commands, name)
}

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

func (c *GomCLI) StartWithInput(input string) error {
	if err := c.processInput(input); err != nil {
		return err
	}

	return c.Start()
}

func (c *GomCLI) Start() error {
	defer c.End()

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

func (c *GomCLI) End() {
	c.writeHistory()
	c.lr.Close()
}
