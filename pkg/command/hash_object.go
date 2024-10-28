package command

import "flag"

func HashObjectCommand() *Command {
	command := newCommand("hash-object")
	command.Action = func(args []string) error {
		write := *flag.Bool("w", true, "Actually write the object into the database")
		path := *flag.String("path", "", "Read object from <file>")
		objType := *flag.String("type", "", "Object type. Possible values are blob, commit, tag, tree")
		return nil
	}
	command.Description = func() string { return "Compute object ID and optionally creates a blob from a file" }
	return command
}
