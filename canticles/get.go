package canticles

import (
	"flag"
	"fmt"
	"log"
)

type Get struct {
	flags      *flag.FlagSet
	Verbose    bool
	Nodeps     bool
	LocalsOnly bool
	Source     string
}

func NewGet() *Get {
	f := flag.NewFlagSet("get", flag.ExitOnError)
	g := &Get{flags: f}
	f.BoolVar(&g.Verbose, "v", false, "Be verbose when getting stuff")
	f.BoolVar(&g.Nodeps, "n", false, "Only fetch the target package, do not resolve deps")
	f.BoolVar(&g.LocalsOnly, "l", false, "Do not attempt to resolve a remote repo for this dep")
	f.StringVar(&g.Source, "source", "", "Overide the VCS url to fetch this from")
	return g
}

var get = NewGet()

var GetCommand = &Command{
	Name:             "get",
	UsageLine:        "get [-v]",
	ShortDescription: "download dependencies as defined in the Canticle file",
	LongDescription: `The get command fetches dependencies. When issued locally it looks...

Specify -v to print out a verbose set of operations instead of just errors.

Specify -n to only download the target package and to not resolve target deps. 

Specify -l to fetch no remote deps but do set any existing repos to the correct revision.`,
	Flags: get.flags,
	Cmd:   get,
}

// Run the get command. Ignores args.
func (g *Get) Run(args []string) {
	if g.Verbose {
		Verbose = true
		defer func() { Verbose = false }()
	}

	pkgArgs := g.flags.Args()
	if g.Source != "" && len(pkgArgs) > 1 {
		log.Fatal("cant get may not be run with -source and multiple packages")
	}
	pkgs := ParseCmdLinePackages(pkgArgs)
	for _, pkg := range pkgs {
		if err := g.GetPackage(pkg); err != nil {
			log.Fatal(err)
		}
	}
}

// Psuedocode
// Load Canticle deps
// Fetch canticle deps
// Walk the dep tree and fetch anything else
// Done

// GetPackage fetches a package and all of it dependencies to either
// the buildroot or the gopath.
func (g *Get) GetPackage(path string) error {
	LogVerbose("Fetching path %+v", path)
	gopath := EnvGoPath()
	resolvers := []RepoResolver{
		&LocalRepoResolver{LocalPath: gopath},
		&RemoteRepoResolver{gopath},
		&DefaultRepoResolver{gopath},
	}
	resolver := NewMemoizedRepoResolver(&CompositeRepoResolver{resolvers})
	depReader := &DepReader{gopath}

	loader := &CanticleDepLoader{
		Reader:   depReader,
		Resolver: resolver,
		Gopath:   gopath,
	}
	if err := loader.FetchPath(path); err != nil {
		return fmt.Errorf("cant load package %s", err.Error())
	}
	return nil

	/*
		// Setup our resolvers, loaders, and walkers
		dl := NewDependencyLoader(resolver, depReader, gopath, path)
		dw := NewDependencyWalker(dl.PackagePaths, dl.FetchUpdatePath)

		// And walk it
		err := dw.TraverseDependencies(path)
		if err != nil {
			return fmt.Errorf("cant fetch packages %s", err.Error())
		}
	*/
}
