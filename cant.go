package main

import (
	"flag"
	"fmt"
	"os"

	"github.comcast.com/viper-cog/cant/canticle"
)

// Canticle will:
// Load deps files
// Make the working dir _workspace
// Grab the deps listed in the file (all deps must be listed), it may grab from an alternative url
// Import the deps
// No rewrite necessary..., just overload gopath
// Copy your current _workspace into in the correct place
// Call build/test/whatever
// Copy the result back out
func main() {
	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		usage()
	}

	cmdName := args[0]
	cmd, ok := canticle.Commands[cmdName]
	if !ok {
		fmt.Fprintln(os.Stderr, "Unkown subcommand ", cmdName)
		usage()
	}

	fmt.Printf("Executing CMD: %+v CMDName: %+v\n", cmd, cmdName)
	cmd.Flags.Usage = cmd.Usage
	cmd.Flags.Parse(args[1:])
	cmd.Cmd.Run(args[1:])
}

var UsageTemplate = `Canticle is a tool for managing go dependencies.

Usage:
  cant command [arguments]

The commands are:
{{range .}}
         {{.Name | printf "%-11s"}} {{.ShortDescription}}{{end}}

Use "cant help [command]" for more information about that command.


`

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: ")
	os.Exit(2)
}
