package canticle

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"path"
)

// Build
type Build struct {
	flags        *flag.FlagSet
	Gopath       string
	InSource     bool
	Verbose      bool
	PreferLocals bool
	GenBuildInfo bool
}

func NewBuild() *Build {
	f := flag.NewFlagSet("build", flag.ExitOnError)
	b := &Build{flags: f, GenBuildInfo: true, Gopath: EnvGoPath()}
	f.BoolVar(&b.InSource, "insource", false, "Get the packages to the enviroment gopath rather than the build dir")
	f.BoolVar(&b.Verbose, "v", false, "Be verbose when getting stuff")
	f.BoolVar(&b.PreferLocals, "l", false, "Prefer local copies from the $GOPATH when getting stuff")
	f.BoolVar(&b.GenBuildInfo, "-gen-buildinfo", true, "Generate buildinfo.go for the package")
	return b
}

var b = NewBuild()

// BuildCommand
var BuildCommand = &Command{
	Name:             "build",
	UsageLine:        `build [-insource] [-v] [-l] [-gen-buildinfo] [package...]`,
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
	pkgs := ParseCmdLinePackages(b.flags.Args())
	LogVerbose("Build packages: %+v", pkgs)
	for _, pkg := range pkgs {
		if err := b.BuildPackage(pkg); err != nil {
			log.Fatalf("Error %s building pkg %s", err.Error(), pkg)
		}
	}
}

func (b *Build) BuildPackage(pkg string) error {
	LogVerbose("Building package %s", pkg)
	// Setup our getter and grab our deps deps
	g := NewGet()
	g.Verbose = b.Verbose
	g.InSource = b.InSource
	g.PreferLocals = b.PreferLocals
	if err := g.GetPackage(pkg); err != nil {
		return err
	}

	if b.GenBuildInfo {
		if err := b.WriteVersion(pkg); err != nil {
			return err
		}
	}

	// And build it
	br := BuildRoot(b.Gopath, pkg)
	cmd := exec.Command("go", "build", pkg)
	if !b.InSource {
		cmd.Env = PatchEnviroment(os.Environ(), "GOPATH", br)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return err
	}

	LogVerbose("Built package %s", pkg)
	return nil
}

func (b *Build) WriteVersion(pkg string) error {
	LogVerbose("Writing version info for package %s", pkg)
	gopath := b.Gopath
	if !b.InSource {
		gopath = BuildRoot(b.Gopath, pkg)
	}
	packageInfo, err := LoadPackage(pkg, gopath)
	if err != nil {
		return err
	}
	// Don't generate buildinfo.go for main packages
	if packageInfo.Name != "main" {
		return nil
	}

	bi, err := NewBuildInfo(gopath, pkg)
	if err != nil {
		return err
	}
	f, err := os.Create(path.Join(PackageSource(gopath, pkg), "generatedbuildinfo.go"))
	if err != nil {
		return err
	}
	defer f.Close()
	return bi.WriteGoFile(f)
}
