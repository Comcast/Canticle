package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"text/template"

	"github.comcast.com/viper-cog/cant/canticles"
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
	versionFlag := flag.Bool("version", false, "version prints the version info of canticle")
	flag.Usage = usage
	flag.Parse()
	log.SetFlags(0)

	if *versionFlag {
		b, err := json.MarshalIndent(BuildInfo, "", "    ")
		if err != nil {
			log.Fatalf("Error marshaling own buildinfo!: %s", err.Error())
		}
		log.Printf("BuildInfo: \n%s\n", string(b))
		return
	}

	args := flag.Args()
	if len(args) < 1 {
		usage()
	}

	cmdName := args[0]
	cmd, ok := canticles.Commands[cmdName]
	if !ok {
		fmt.Fprintln(os.Stderr, "Unkown subcommand ", cmdName)
		usage()
	}

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
	tmpl, _ := template.New("UsageTemplate").Parse(UsageTemplate)
	tmpl.Execute(os.Stderr, canticles.Commands)
	os.Exit(2)
}
