package canticles

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"path"
)

type Save struct {
	flags   *flag.FlagSet
	Verbose bool
}

func NewSave() *Save {
	f := flag.NewFlagSet("get", flag.ExitOnError)
	s := &Save{flags: f}
	f.BoolVar(&s.Verbose, "v", false, "Be verbose when getting stuff")

	return s
}

var save = NewSave()

var SaveCommand = &Command{
	Name:             "save",
	UsageLine:        "save [-v] [package]",
	ShortDescription: "Save the current revision of all dependencies in a Canticle file.",
	LongDescription: `The save command will save the dependencies for a package into a Canticle file. All dependencies must be present on disk and in the GOROOT. The generate Canticle file will be saved in the packages root directory.

Specify -v to print out a verbose set of operations instead of just errors.`,
	Flags: save.flags,
	Cmd:   save,
}

func (s *Save) Run(args []string) {
	if s.Verbose {
		Verbose = true
	}
	defer func() { Verbose = false }()

	pkgs := ParseCmdLinePackages(s.flags.Args())
	for _, pkg := range pkgs {
		ldeps, err := s.LocalDeps(pkg)
		if err != nil {
			log.Fatal(err)
		}
		err = s.SaveDeps(pkg, ldeps)
		if err != nil {
			log.Fatal(err)
		}
	}
}

// LocalDeps reads a packages dependencies on disk an there
// versions and remotes.
func (s *Save) LocalDeps(pkg string) (Dependencies, error) {
	LogVerbose("Save dependencies for package %+v", pkg)
	gopath := EnvGoPath()

	// Setup our resolvers, loaders, and walkers
	lr := &LocalRepoResolver{LocalPath: gopath}
	resolver := NewMemoizedRepoResolver(lr)
	depReader := &DepReader{gopath}
	ds := NewDependencySaver(resolver, depReader.ReadAllCantDeps, gopath, pkg)
	dw := NewDependencyWalker(depReader.ReadAllRemoteDependencies, ds.SavePackageRevision)

	// And walk it
	err := dw.TraverseDependencies(pkg)
	if err != nil {
		return nil, fmt.Errorf("cant save package %s", err.Error())
	}
	LogVerbose("Package %s has remotes: %+v", pkg, ds.Dependencies())

	return ds.Dependencies(), nil
}

// SaveDeps saves a canticle file into deps containing deps.
func (s *Save) SaveDeps(pkg string, deps Dependencies) error {
	j, err := json.MarshalIndent(deps, "", "    ")
	if err != nil {
		return err
	}

	cantFile := path.Join(PackageSource(EnvGoPath(), pkg), "Canticle")
	return ioutil.WriteFile(cantFile, j, 0644)
}
