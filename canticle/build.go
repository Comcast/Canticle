package canticle

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
)

// Build
type Build struct {
	flags        *flag.FlagSet
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
	g := NewGet()
	if b.Verbose {
		Verbose = true
	}
	defer func() { Verbose = false }()
	g.Verbose = b.Verbose
	g.Insource = b.Insource
	g.PreferLocals = b.PreferLocals

	gopath := EnvGoPath()
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error getting working directory: %s", err.Error())
	}

	deps := ParseCmdLineDeps(b.flags.Args())
	LogVerbose("Deps: %+v", deps)
	for _, dep := range deps {
		LogVerbose("Building dep %s", dep.ImportPath)
		// Grab its deps
		g.GetPackage(dep)
		// And build it
		br := BuildRoot(gopath, dep.ImportPath)
		LogVerbose("Building at buildroot: %s", br)
		if err := os.Chdir(br); err != nil {
			log.Fatalf("Unable to chdir to buildroot %s error %s", br, err.Error())
		}
		cmd := exec.Command("go", "build")
		cmd.Dir = PackageDir(gopath, dep.ImportPath)
		cmd.Env = PatchEnviroment(os.Environ(), "GOPATH", br)
		var sout, serr bytes.Buffer
		cmd.Stdout = &sout
		cmd.Stderr = &serr
		err := cmd.Run()
		fmt.Println((&sout).String())
		fmt.Fprintln(os.Stderr, (&serr).String())
		if err != nil {
			log.Fatalf("Error building dep %s", dep.ImportPath)
		} else {
			LogVerbose("Built package %s", dep.ImportPath)
		}
		restoreWD(cwd)
	}

}
