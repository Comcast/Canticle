package canticle

import (
	"flag"
	"log"
	"os"
	"os/exec"
)

// Build
type Build struct {
	flags        *flag.FlagSet
	Gopath       string
	Insource     bool
	Verbose      bool
	PreferLocals bool
}

func NewBuild() *Build {
	f := flag.NewFlagSet("build", flag.ExitOnError)
	b := &Build{flags: f}
	f.BoolVar(&b.Insource, "insource", false, "Get the packages to the enviroment gopath rather than the build dir")
	f.BoolVar(&b.Verbose, "v", false, "Be verbose when getting stuff")
	f.BoolVar(&b.PreferLocals, "l", false, "Prefer local copies from the $GOPATH when getting stuff")
	return b
}

var b = NewBuild()

// BuildCommand
var BuildCommand = &Command{
	Name:             "build",
	UsageLine:        `build [-insource] [-v] [-l] [package,<source>...]`,
	ShortDescription: `download dependencies as defined in the packages Canticle file and build the project`,
	LongDescription: `The build command first gets the packages (see cant get help). The build command may be used against both non Canticle defined (no revisions wil be set) and Canticle defined packages.

If -insource is specified only one package may be specified. Instead packages will be fetched into the $GOPATH as necessary and set to the correct revision.  

Specify -l to prefer local copies from $GOPATH when trying to fetch a package for building.

Specify -v to print out a verbose set of operations instead of just errors.
`,
	Flags: b.flags,
	Cmd:   b,
}

// Run
func (b *Build) Run(args []string) {
	if b.Verbose {
		Verbose = true
	}
	defer func() { Verbose = false }()
	b.Gopath = EnvGoPath()
	deps := ParseCmdLineDeps(b.flags.Args())
	LogVerbose("Deps: %+v", deps)
	for _, dep := range deps {
		if err := b.BuildPackage(dep); err != nil {
			log.Fatalf("Error %s building dep %s", err.Error(), dep.ImportPath)
		}
	}
}

func (b *Build) BuildPackage(dep *Dependency) error {
	LogVerbose("Building dep %s", dep.ImportPath)
	// Setup our getter and grab our deps deps
	g := NewGet()
	g.Verbose = b.Verbose
	g.Insource = b.Insource
	g.PreferLocals = b.PreferLocals
	g.GetPackage(dep)

	// And build it
	br := BuildRoot(b.Gopath, dep.ImportPath)
	cmd := exec.Command("go", "build", dep.ImportPath)
	cmd.Env = PatchEnviroment(os.Environ(), "GOPATH", br)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return err
	}

	LogVerbose("Built package %s", dep.ImportPath)
	return nil
}
