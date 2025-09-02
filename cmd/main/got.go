package main

import (
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/jessegeens/got/pkg/command"
)

var (
	commands = []*command.Command{
		command.AddCommand(),
		command.CatFileCommand(),
		command.CheckIgnoreCommand(),
		command.CheckoutCommand(),
		command.CommitCommand(),
		command.HashObjectCommand(),
		command.InitCommand(),
		command.LogCommand(),
		command.LsFilesCommand(),
		command.LsTreeCommand(),
		command.RevParseCommand(),
		command.RmCommand(),
		command.ShowRefCommand(),
		command.StatusCommand(),
		command.TagCommand(),
	}
)

func init() {}

func main() {
	if len(os.Args) < 2 {
		os.Exit(1)
	}
	args := os.Args[1:]
	commandName := args[0]
	if commandName == "--help" || commandName == "-h" {
		printHelp()
	}

	flag.Parse()
	for _, command := range commands {
		if command.Name == commandName {
			// Now, we remove the command from the args list, because
			// the `flags` package stops parsing after the first non-option
			os.Args = []string{os.Args[0]}
			os.Args = append(os.Args, args[1:]...)

			err := command.Action(args[1:])
			if err != nil {
				fmt.Printf("Failed to execute command %s with error:\n\t %s\n", commandName, err.Error())
				os.Exit(1)
			}
			os.Exit(0)
		}
	}
	fmt.Printf("got: '%s' is not a got command. See 'got --help'\n", commandName)
	os.Exit(1)
}

func printHelp() {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)

	// Print header
	fmt.Fprintln(w, "Command\tDescription")
	for _, cmd := range commands {
		fmt.Fprintf(w, "%s\t%s\n", cmd.Name, cmd.Description())
	}

	w.Flush()

	os.Exit(0)
}
