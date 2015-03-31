package canticle

import (
	"fmt"
)

// Build
type Build struct {
}

// BuildCommand
var BuildCommand = &Command{
	Name:             "build",
	UsageLine:        ``,
	ShortDescription: ``,
	LongDescription:  ``,
	Cmd:              &Build{},
}

// Run
func (b *Build) Run(args []string) {
	fmt.Printf("Run")
}
