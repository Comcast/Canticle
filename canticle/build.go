package canticle

import (
	"flag"
	"fmt"
)

// Build
type Build struct {
	flags *flag.FlagSet
}

func NewBuild() *Build {
	f := flag.NewFlagSet("build", flag.ExitOnError)
	b := &Build{flags: f}
	return b
}

var b = NewBuild()

// BuildCommand
var BuildCommand = &Command{
	Name:             "build",
	UsageLine:        ``,
	ShortDescription: ``,
	LongDescription:  ``,
	Flags:            b.flags,
	Cmd:              b,
}

// Run
func (b *Build) Run(args []string) {
	fmt.Printf("Run")
}
