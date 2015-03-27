package canticle

import "flag"

type CanticleCommand interface {
	Run(args []string)
}

// Command represents a canticle command to be run including:
// *  Save
// *  Build
// *  Test
// *  Update
type Command struct {
	UsageLine        string
	ShortDescription string
	LongDescription  string
	Flags            flag.FlagSet
	Cmd              CanticleCommand
}

var CanticleCommands = map[string]*Command{
	"build": BuildCommand,
	"get":   GetCommand,
}
