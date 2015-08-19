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
	UsageLine:        "get [-source <source>] [-v] [-n] [-l] [package...]",
	ShortDescription: "download dependencies as defined in the Canticle file",
	LongDescription: `The get command fetches dependencies. When issued locally it looks...
If get is issued against a package which does not exist it will also be downloaded.

If -source is specified the package will be fetched from this specified vcs repo source.

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

// GetPackage fetches a package and all of it dependencies to either
// the buildroot or the gopath.
func (g *Get) GetPackage(pkg string) error {
	LogVerbose("Fetching package %+v", pkg)
	gopath := EnvGoPath()
	// Setup our resolvers, loaders, and walkers
	resolvers := []RepoResolver{
		&LocalRepoResolver{LocalPath: gopath},
		&RemoteRepoResolver{gopath},
		&DefaultRepoResolver{gopath},
	}
	resolver := NewMemoizedRepoResolver(&CompositeRepoResolver{resolvers})
	depReader := &DepReader{gopath}
	dl := NewDependencyLoader(resolver, depReader, gopath, PackageSource(gopath, pkg))
	dw := NewDependencyWalker(depReader.AllImports, dl.FetchUpdatePath)

	// And walk it
	err := dw.TraverseDependencies(pkg)
	if err != nil {
		return fmt.Errorf("cant fetch packages %s", err.Error())
	}

	return nil
}
