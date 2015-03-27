package main

import (
	"fmt"
	"os"

	"github.comcast.com/viper-cog/canticle/canticle"
)

var usage = "cant "

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
	if len(os.Args) < 2 {
		fmt.Println(usage)
		return
	}

	cmdName := os.Args[1]
	cmd, ok := canticle.CanticleCommands[cmdName]
	if !ok {
		fmt.Println("Unkown canticle command")
	}

	fmt.Printf("Executing CMD: %+v CMDName: %+v\n", cmd, cmdName)
	cmd.Cmd.Run(os.Args[2:])
}
