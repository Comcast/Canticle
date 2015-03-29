package canticle

import (
	"fmt"
)

type Build struct {
}

var BuildCommand = &Command{
	Name:             "build",
	UsageLine:        ``,
	ShortDescription: ``,
	LongDescription:  ``,
	Cmd:              &Build{},
}

func (b *Build) Run(args []string) {
	fmt.Printf("Run")
}
