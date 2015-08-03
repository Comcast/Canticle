package canticles

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

type Save struct {
	flags    *flag.FlagSet
	Verbose  bool
	DryRun   bool
	Force    bool
	Resolver ConflictResolver
}

func NewSave() *Save {
	f := flag.NewFlagSet("get", flag.ExitOnError)
	s := &Save{
		flags:    f,
		Resolver: &PromptResolution{},
	}
	f.BoolVar(&s.Verbose, "v", false, "Be verbose when getting stuff")
	f.BoolVar(&s.Verbose, "f", false, "Force a save even when there are missing deps or version conflicts.")
	f.BoolVar(&s.DryRun, "d", false, "Don't save the deps, just print them.")
	return s
}

var save = NewSave()

var SaveCommand = &Command{
	Name:             "save",
	UsageLine:        "save [-f] [-d] [-v]",
	ShortDescription: "Save the current revision of all dependencies in a Canticle file.",
	LongDescription: `The save command will save the dependencies for a package into a Canticle file.  If at the src level save the current revision of all packages in belows. All dependencies must be present on disk and in the GOROOT. The generated Canticle file will be saved in the packages root directory.

Specify -v to print out a verbose set of operations instead of just errors.

Specify -f to force a save even with missing dependencies or conflicting versions.`,
	Flags: save.flags,
	Cmd:   save,
}

// Run the save command, ignores args. Uses its flagset instead.
func (s *Save) Run(args []string) {
	if s.Verbose {
		Verbose = true
	}
	defer func() { Verbose = false }()

	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	if err := s.SaveProject(wd); err != nil {
		log.Fatal(err)
	}
}

func (s *Save) SaveProject(path string) error {
	gopath := EnvGoPath()
	deps, err := s.ReadDeps(gopath, path)
	if err != nil {
		return err
	}
	sources, err := s.GetSources(gopath, path, deps)
	if err != nil {
		return err
	}
	LogVerbose("Discovered sources:\n%+v", sources)
	cantdeps, err := s.Resolver.ResolveConflicts(sources)
	if err != nil {
		return err
	}
	if err := s.SaveDeps(path, cantdeps); err != nil {
		return err
	}
	return nil
}

func (s *Save) GetSources(gopath, path string, deps Dependencies) (*DependencySources, error) {
	LogVerbose("Getting local vcs sources for repos in path %+v", gopath)
	repoResolver := NewMemoizedRepoResolver(&LocalRepoResolver{gopath})
	sourceResolver := &SourcesResolver{
		Gopath:   gopath,
		RootPath: path,
		Resolver: repoResolver,
	}
	return sourceResolver.ResolveSources(deps)
}

// ReadDeps reads all dependencies and transitive deps for deps. May
// mutate deps.
func (s *Save) ReadDeps(gopath, path string) (Dependencies, error) {
	LogVerbose("Reading deps for repos in path %+v", gopath)
	reader := &DepReader{Gopath: gopath}
	ds := NewDependencySaver(reader.ReadAllDeps, gopath)
	dw := NewDependencyWalker(ds.PackagePaths, ds.SavePackageDeps)
	if err := dw.TraverseDependencies(path); err != nil {
		return nil, fmt.Errorf("cant read path dep tree %s %s", path, err.Error())
	}
	LogVerbose("Built dep tree: %+v", ds.Dependencies())
	return ds.Dependencies(), nil
}

// SaveDeps saves a canticle file into deps containing deps.
func (s *Save) SaveDeps(path string, deps []*CanticleDependency) error {
	j, err := json.MarshalIndent(deps, "", "    ")
	if err != nil {
		return err
	}
	if s.DryRun {
		fmt.Println(string(j))
		return nil
	}
	return ioutil.WriteFile(DependencyFile(path), j, 0644)
}

// Psuedo Code:
// save set of repos
// get set of deps
// report conflicts
// save
