package canticles

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
)

type Save struct {
	flags    *flag.FlagSet
	Verbose  bool
	DryRun   bool
	OnDisk   bool
	Branches bool
	Excludes DirFlags
	Resolver ConflictResolver
}

func NewSave() *Save {
	f := flag.NewFlagSet("save", flag.ExitOnError)
	s := &Save{
		flags:    f,
		Resolver: &PromptResolution{},
		Excludes: DirFlags(NewStringSet()),
	}
	f.BoolVar(&s.Verbose, "v", false, "Be verbose when getting stuff")
	f.BoolVar(&s.OnDisk, "ondisk", false, "Save the revisions and sources present on disk ignoring all other Canticle files.")
	f.BoolVar(&s.DryRun, "d", false, "Don't save the deps, just print them.")
	f.BoolVar(&s.Branches, "b", false, "Save branches for the current projects, not revisions.")
	f.Var(&s.Excludes, "exclude", "Do not recur into these directories when saving unless they are in the dep tree.")
	return s
}

var save = NewSave()

var SaveCommand = &Command{
	Name:             "save",
	UsageLine:        "save [-f] [-ondisk] [-d] [-b] [-exclude <dir>] ",
	ShortDescription: "Save the current revision of all dependencies in a Canticle file.",
	LongDescription: `The save command will save the dependencies for a package into a Canticle file.  If at the src level save the current revision of all packages in belows. All dependencies must be present on disk and in the GOROOT. The generated Canticle file will be saved in the packages root directory.

Specify -v to print out a verbose set of operations instead of just errors.

Specify -ondisk to use on disk revisions and sources and do no conflict resolution.

Specify -b to save branches or tags when present instead of revisions`,
	Flags: save.flags,
	Cmd:   save,
}

// Run the save command, ignores args. Uses its flagset instead.
func (s *Save) Run(args []string) {
	if s.Verbose {
		Verbose = true
	}
	defer func() { Verbose = false }()
	if s.OnDisk {
		s.Resolver = &PreferLocalResolution{}
	}

	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	if err := s.SaveProject(wd); err != nil {
		log.Fatal(err)
	}
}

// SaveProject does four things:
//   *  It fetches the dep tree of path
//   *  It fetches all possible DependencySources
//   *  It performs conflict resolution
//   *  It saves a Canticle file in path
func (s *Save) SaveProject(path string) error {
	gopath, err := EnvGoPath()
	if err != nil {
		return err
	}
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

// GetSources returns the DependencySources (e.g. the possible revisions, vcs sources, and deps)
// for a give path, and set Dependencies.
func (s *Save) GetSources(gopath, path string, deps Dependencies) (*DependencySources, error) {
	LogVerbose("Getting local vcs sources for repos in path %+v", gopath)
	repoResolver := NewMemoizedRepoResolver(&LocalRepoResolver{gopath})
	reader := &DepReader{Gopath: gopath}
	sourceResolver := &SourcesResolver{
		Gopath:     gopath,
		RootPath:   path,
		Resolver:   repoResolver,
		Branches:   s.Branches,
		CDepReader: reader,
	}
	return sourceResolver.ResolveSources(deps)
}

// ReadDeps reads all dependencies and transitive deps for path.
func (s *Save) ReadDeps(gopath, path string) (Dependencies, error) {
	LogVerbose("Reading deps for repos in path %+v", gopath)
	reader := &DepReader{Gopath: gopath}
	ds := NewDependencySaver(reader.AllDeps, gopath, path)
	ds.NoRecur = StringSet(s.Excludes)
	dw := NewDependencyWalker(ds.PackagePaths, ds.SavePackageDeps)
	if err := dw.TraverseDependencies(path); err != nil {
		return nil, fmt.Errorf("cant read path dep tree %s %s", path, err.Error())
	}
	LogVerbose("Built dep tree: %+v", ds.Dependencies())
	return ds.Dependencies(), nil
}

// SaveDeps saves a canticle file at path containing deps.
func (s *Save) SaveDeps(path string, deps []*CanticleDependency) error {
	sort.Sort(CanticleDependencies(deps))
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
