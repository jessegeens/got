package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/jessegeens/go-toolbox/pkg/command"
)

var (
	commands = []*command.Command{
		command.InitCommand(),
		command.CatFileCommand(),
	}
)

func init() {}

func main() {
	if len(os.Args) < 2 {
		os.Exit(1)
	}
	flag.Parse()
	args := os.Args[1:]
	commandName := args[0]
	for _, command := range commands {
		if command.Name == commandName {
			err := command.Action(args[1:])
			if err != nil {
				fmt.Printf("Failed to execute command %s with error %s\n", commandName, err.Error())
			}
		}
	}
}
