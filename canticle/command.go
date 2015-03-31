package canticle

import (
	"flag"
	"fmt"
	"os"
)

type Runnable interface {
	Run(args []string)
}

// Command represents a canticle command to be run including:
// *  Save
// *  Build
// *  Test
// *  Update
type Command struct {
	Name             string
	UsageLine        string
	ShortDescription string
	LongDescription  string
	Flags            flag.FlagSet
	Cmd              Runnable
}

var Commands = map[string]*Command{
	"build": BuildCommand,
	"get":   GetCommand,
}

func (c *Command) Usage() {
	fmt.Fprintf(os.Stderr, "usage %s\n", c.UsageLine)
	fmt.Fprintf(os.Stderr, "%s\n", c.LongDescription)
	os.Exit(2)
}
