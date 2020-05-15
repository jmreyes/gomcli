package gomcli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/anmitsu/go-shlex"
	"github.com/peterh/liner"
)

var ErrCliPromptAborted = errors.New("prompt aborted")
var ErrCliCommandNotFound = errors.New("command not found")
var ErrCliCannotParseLine = errors.New("cannot parse line")

type NotFoundHandler func(string) error

type Conf struct {
	Prompt          string
	HistFile        string
	CtrlCAborts     bool
	NotFoundHandler NotFoundHandler
}

type CLI struct {
	lr              *liner.State
	prompt          string
	histfile        string
	commands        map[string]Command
	notFoundHandler NotFoundHandler
}

func New(conf Conf) *CLI {
	cli := &CLI{}
	cli.lr = liner.NewLiner()

	cli.prompt = conf.Prompt
	if cli.prompt == "" {
		cli.prompt = "gomcli > "
	}

	cli.lr.SetCtrlCAborts(conf.CtrlCAborts)
	cli.lr.SetTabCompletionStyle(liner.TabPrints)
	cli.lr.SetWordCompleter(cli.complete)

	cli.notFoundHandler = conf.NotFoundHandler

	cli.histfile = conf.HistFile
	cli.setupHistory()

	cli.commands = make(map[string]Command)

	return cli
}

func (cli *CLI) AddCommand(command Command) {
	cli.commands[command.Name] = command
}

func (cli *CLI) setupHistory() {
	if cli.histfile == "" {
		return
	}

	f, err := os.Open(cli.histfile)
	if err != nil {
		return
	}
	cli.lr.ReadHistory(f)
	f.Close()
}

func (cli *CLI) writeHistory() error {
	if cli.histfile == "" {
		return nil
	}

	dirName := filepath.Dir(cli.histfile)
	if _, err := os.Stat(dirName); err != nil {
		err := os.MkdirAll(dirName, os.ModePerm)
		if err != nil {
			return err
		}
	}

	f, err := os.Create(cli.histfile)
	if err != nil {
		return err
	}
	cli.lr.WriteHistory(f)
	f.Close()

	return nil
}

func (cli *CLI) complete(line string, pos int) (head string, c []string, tail string) {
	tokens, _ := shlex.Split(line[:pos], false)
	tail = line[pos:]
	for i := len(tokens); i > 0; i-- {
		chunk := strings.Join(tokens[:i], " ")
		if cmd, err := cli.getCommand(chunk); err == nil {
			if i == len(tokens) {
				return line, cmd.Complete(""), tail
			}
			search := tokens[i]
			return cmd.Name + " ", cmd.Complete(search), tail
		}
	}
	return head, cli.rawCommandCompleter(line), tail
}

func (cli *CLI) ContextualComplete() []string {
	keys := make([]string, 0, len(cli.commands))
	for k := range cli.commands {
		keys = append(keys, k)
	}
	return keys
}

func (cli *CLI) rawCommandCompleter(line string) (res []string) {
	for _, cmd := range cli.ContextualComplete() {
		if strings.HasPrefix(cmd, line) {
			res = append(res, cmd)
		}
	}
	return
}

func (cli *CLI) getCommand(name string) (*Command, error) {
	if cmd, ok := cli.commands[name]; ok {
		return &cmd, nil
	}
	return nil, ErrCliCommandNotFound
}

func (cli *CLI) process() error {
	userInput, err := cli.lr.Prompt(cli.prompt)
	if err != nil {
		return err
	}

	cli.lr.AppendHistory(userInput)

	return cli.processInput(userInput)
}

func (cli *CLI) processInput(input string) error {

	lines, err := cli.splitInlineCommands(input)
	if err != nil {
		return err
	}

	for _, line := range lines {
		err := cli.processLine(line)
		if err != nil {
			return err
		}
	}

	return nil
}

func (cli *CLI) processLine(line string) error {
	tokens, err := shlex.Split(line, true)
	if err != nil {
		return err
	}

	if len(tokens) == 0 {
		return nil
	}

	for i := len(tokens); i > 0; i-- {
		chunk := strings.Join(tokens[:i], " ")
		cmd, err := cli.getCommand(chunk)
		if err != nil {
			continue
		}

		if len(tokens) > 1 {
			return cmd.Execute(tokens[i:]...)
		}
		return cmd.Execute()
	}

	if cli.notFoundHandler != nil {
		err = cli.notFoundHandler(tokens[0])
	}
	return err
}

func (cli *CLI) splitInlineCommands(userInput string) ([]string, error) {
	parsed, err := shlex.Split(userInput, false)
	if err != nil {
		return nil, err
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

func (cli *CLI) StartWithInput(input string) error {
	if err := cli.processInput(input); err != nil {
		return err
	}

	return cli.Start()
}

func (cli *CLI) Start() error {
	defer cli.End()

	for {
		if err := cli.process(); err != nil {
			switch err {
			case liner.ErrPromptAborted:
				return ErrCliPromptAborted
			default:
				return err
			}
		}
	}
}

func (cli *CLI) End() {
	cli.writeHistory()
	cli.lr.Close()
}
