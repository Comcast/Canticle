package canticles

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
)

type Vendor struct {
	flags    *flag.FlagSet
	Verbose  bool
	Sources  string
	Resolver ConflictResolver
}

func NewVendor() *Vendor {
	f := flag.NewFlagSet("save", flag.ExitOnError)
	s := &Vendor{
		flags:    f,
		Resolver: &PromptResolution{},
	}
	f.BoolVar(&s.Verbose, "v", false, "Be verbose when getting stuff")
	f.StringVar(&s.Sources, "s", "", "Use this canticle file to source repos.")
	return s
}

var vendor = NewVendor()

var VendorCommand = &Command{
	Name:             "vendor",
	UsageLine:        "vendor [-v] [-s sourcefile]",
	ShortDescription: "Download the all dependencies of a project.",
	LongDescription: `The vendor command will download all dependencies of a package in its go and Canticle dependency graph.

Specify -v to print out a verbose set of operations instead of just errors.

Specify -s <filename>, where filename contains Canticle deps to specify alternative sources to fetch packages from.`,
	Flags: vendor.flags,
	Cmd:   vendor,
}

func (v *Vendor) Run(args []string) {
	if v.Verbose {
		Verbose = true
	}
	defer func() { Verbose = false }()

	var deps []*CanticleDependency
	if v.Sources != "" {
		f, err := os.Open(v.Sources)
		if err != nil {
			log.Fatal("cant open dep file %s", v.Sources)
			return
		}
		LogVerbose("Reading canticle file: %s", f.Name())
		defer f.Close()
		d := json.NewDecoder(f)
		if err := d.Decode(&deps); err != nil {
			log.Fatal("cant decode dep file %s", v.Sources)
			return
		}
	}

	for _, pkg := range v.flags.Args() {
		LogWarn("Vendoring package %s", pkg)
		if err := v.Vendor(pkg, deps); err != nil {
			log.Fatal(err)
		}
	}
}

func (v *Vendor) Vendor(pkg string, deps []*CanticleDependency) error {
	LogVerbose("Fetching pkg %+v", pkg)
	gopath := EnvGoPath()
	resolvers := []RepoResolver{
		&LocalRepoResolver{LocalPath: gopath},
		&RemoteRepoResolver{gopath},
		&DefaultRepoResolver{gopath},
	}
	resolver := NewMemoizedRepoResolver(&CompositeRepoResolver{resolvers})
	depReader := &DepReader{gopath}

	// Setup our resolvers, loaders, and walkers
	dl := NewDependencyLoader(resolver, depReader.AllDeps, deps, gopath)
	dw := NewDependencyWalker(dl.PackageImports, dl.FetchUpdatePackage)

	// And walk it
	err := dw.TraverseDependencies(pkg)
	if err != nil {
		return fmt.Errorf("cant fetch packages %s", err.Error())
	}

	return nil
}
