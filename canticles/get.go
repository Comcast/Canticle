package canticles

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
)

type Get struct {
	flags   *flag.FlagSet
	Verbose bool
	Update  bool
	Source  string
}

func NewGet() *Get {
	f := flag.NewFlagSet("get", flag.ExitOnError)
	g := &Get{flags: f}
	f.BoolVar(&g.Verbose, "v", false, "Be verbose when getting stuff")
	f.BoolVar(&g.Update, "u", false, "Update branches where possible, print the results")
	f.StringVar(&g.Source, "source", "", "Overide the VCS url to fetch this from")
	return g
}

var get = NewGet()

var GetCommand = &Command{
	Name:             "get",
	UsageLine:        "get [-v] [-u]",
	ShortDescription: "download dependencies as defined in the Canticle file",
	LongDescription: `The get command fetches dependencies. When issued locally it looks...

Specify -v to print out a verbose set of operations instead of just errors.

Specify -u to update branches and print results.`,
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
	gopath, err := EnvGoPath()
	if err != nil {
		return err
	}
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
		Update:   g.Update,
	}
	if errs := loader.FetchPath(path); len(errs) > 0 {
		for _, err := range errs {
			return fmt.Errorf("cant load package %s", err.Error())
		}
	}
	if g.Update {
		b, err := json.Marshal(loader.Updated())
		if err != nil {
			return err
		}
		fmt.Print("Updated packages: ", string(b))
	}
	return nil
}
