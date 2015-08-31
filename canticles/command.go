package canticles

import (
	"flag"
	"fmt"
	"log"
	"os"
)

// Runnable is used by Command to call an item with the arguments
// pertinent to it.
type Runnable interface {
	Run(args []string)
}

// Command represents a Canticle command to be run including:
// *  Save
// *  Build
// *  Test
// *  Update
type Command struct {
	Name             string
	UsageLine        string
	ShortDescription string
	LongDescription  string
	Flags            *flag.FlagSet
	Cmd              Runnable
}

// Commands is the prebuild list of Canticle commands.
var Commands = map[string]*Command{
	"get":    GetCommand,
	"save":   SaveCommand,
	"vendor": VendorCommand,
}

// Usage will print the commands UsageLine and LongDescription and
// then os.Exit(2).
func (c *Command) Usage() {
	fmt.Fprintf(os.Stderr, "usage %s\n", c.UsageLine)
	fmt.Fprintf(os.Stderr, "%s\n", c.LongDescription)
	os.Exit(2)
}

// GetCurrentPackage returns the "package name" of the current working
// directory if you the cwd is in the goroot.
func GetCurrentPackage() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	return PackageName(EnvGoPath(), cwd)
}

// ParseCmdLineDeps parses an array of packages or grabs the current
// package if none are present.
func ParseCmdLinePackages(args []string) []string {
	if len(args) == 0 {
		pkg, err := os.Getwd()
		if err != nil {
			log.Fatalf("cant get current package: %s", err.Error())
		}
		return []string{pkg}
	}
	return args
}
