package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/shlex"
	"github.com/peterh/liner"
)

type CLIConf struct {
	Prompt   string
	Banner   string
	HistFile string
}

type CLI struct {
	lr       *liner.State
	prompt   string
	banner   string
	commands map[string]Command
}

func NewSecCLI(conf CLIConf) *CLI {
	cli := &CLI{}
	cli.lr = liner.NewLiner()
	cli.lr.SetCtrlCAborts(true)
	cli.prompt = conf.Prompt
	cli.banner = conf.Banner
	cli.commands = make(map[string]Command)

	cli.setupHistory(conf.HistFile)
	cli.lr.SetCompleter(func(line string) (c []string) {
		return cli.complete(line)
	})

	return cli
}

func (cli *CLI) AddCommand(command Command) {
	cli.commands[command.name] = command
}

func (cli *CLI) setupHistory(path string) {
	dirName := filepath.Dir(path)
	if _, err := os.Stat(dirName); err != nil {
		err := os.MkdirAll(dirName, os.ModePerm)
		if err != nil {
			panic(err)
		}
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	cli.lr.ReadHistory(f)
}

func (cli *CLI) parseLine(line string) ([]string, error) {
	res, err := shlex.Split(line)
	if err != nil {
		return nil, &commandError{err: fmt.Sprintf("Can't parse line: %s", line)}
	}
	return res, nil
}

func (cli *CLI) complete(line string) []string {
	if len(line) > 0 {
		cmd, _ := cli.parseLine(line)
		command, _ := cli.getCommand(cmd[0])
		if command != nil {
			return command.Complete(line)
		}
	}
	return cli.rawCommandCompleter(line)
}

func (cli *CLI) ContextualComplete() []string {
	keys := make([]string, 0, len(cli.commands))
	for k := range cli.commands {
		keys = append(keys, k)
	}
	return keys
}

func (cli *CLI) rawCommandCompleter(line string) (res []string) {
	for _, command := range cli.ContextualComplete() {
		if strings.HasPrefix(command, line) {
			res = append(res, command)
		}
	}
	return
}

func (cli *CLI) getCommand(name string) (*Command, error) {
	if command, ok := cli.commands[name]; ok {
		return &command, nil
	}
	return nil, &commandError{err: fmt.Sprintf("Unknown command: %s", name)}
}

func (cli *CLI) process() error {
	userInput, err := cli.lr.Prompt(cli.prompt)
	if err == nil {
		// TODO _split_inline_commands
		tokens, err := cli.parseLine(userInput)
		if err != nil {
			return err
		}
		if len(tokens) == 0 {
			return nil
		}
		if command, err := cli.getCommand(tokens[0]); err == nil {
			cli.lr.AppendHistory(userInput)
			if len(tokens) > 1 {
				command.Execute(tokens[1:]...)
			} else {
				command.Execute()
			}
		}
	}
	return err
}

func splitInlineCommands(userInput string) {

}

func (cli *CLI) Start() {
	fmt.Printf(cli.banner)

	for {
		err := cli.process()
		if err == liner.ErrPromptAborted {
			break
		}
	}

	cli.lr.Close()
}
