package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/shlex"
	"github.com/peterh/liner"
)

var ErrCliPromptAborted = errors.New("prompt aborted")
var ErrCliCommandNotFound = errors.New("command not found")
var ErrCliCannotParseLine = errors.New("cannot parse line")

type NotFoundHandler func(string) error

type CLIConf struct {
	Prompt      string
	Banner      string
	HistFile    string
	CtrlCAborts bool
}

type CLI struct {
	lr              *liner.State
	prompt          string
	banner          string
	histfile        string
	commands        map[string]Command
	notFoundHandler NotFoundHandler
}

func NewSecCLI(conf CLIConf) *CLI {
	cli := &CLI{}
	cli.lr = liner.NewLiner()
	cli.prompt = conf.Prompt
	if cli.prompt == "" {
		cli.prompt = "gomcli > "
	}
	cli.banner = conf.Banner
	cli.lr.SetCtrlCAborts(conf.CtrlCAborts)
	cli.commands = make(map[string]Command)

	cli.lr.SetTabCompletionStyle(liner.TabPrints)

	cli.histfile = conf.HistFile
	cli.setupHistory()
	cli.lr.SetWordCompleter(cli.complete)

	return cli
}

func (cli *CLI) AddCommand(command Command) {
	cli.commands[command.name] = command
}

func (cli *CLI) setupHistory() {
	if f, err := os.Open(cli.histfile); err == nil {
		cli.lr.ReadHistory(f)
		f.Close()
	}
}

func (cli *CLI) writeHistory() {
	dirName := filepath.Dir(cli.histfile)
	if _, err := os.Stat(dirName); err != nil {
		err := os.MkdirAll(dirName, os.ModePerm)
		if err != nil {
			panic(err)
		}
	}

	if f, err := os.Create(cli.histfile); err != nil {
		panic(err)
	} else {
		cli.lr.WriteHistory(f)
		f.Close()
	}
}

func (cli *CLI) parseLine(line string) ([]string, error) {
	res, err := shlex.Split(line)
	if err != nil {
		return nil, ErrCliCannotParseLine
	}
	return res, nil
}

func (cli *CLI) complete(line string, pos int) (head string, c []string, tail string) {
	tokens, _ := cli.parseLine(line[:pos])
	tail = line[pos:]
	for i := len(tokens); i > 0; i-- {
		chunk := strings.Join(tokens[:i], " ")
		if cmd, err := cli.getCommand(chunk); err == nil {
			if i == len(tokens) {
				return line, cmd.Complete(""), tail
			}
			search := tokens[i]
			return cmd.name + " ", cmd.Complete(search), tail
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
	// TODO _split_inline_commands
	tokens, err := cli.parseLine(userInput)
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

		cli.lr.AppendHistory(userInput)

		if len(tokens) > 1 {
			err = cmd.Execute(tokens[i:]...)
		} else {
			err = cmd.Execute()
		}
		if err != nil {
			return err
		}
		return nil
	}

	if cli.notFoundHandler != nil {
		err = cli.notFoundHandler(tokens[0])
	}
	return err
}

func splitInlineCommands(userInput string) {

}

func (cli *CLI) Start() error {
	defer cli.lr.Close()
	defer cli.writeHistory()

	fmt.Printf(cli.banner)

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
