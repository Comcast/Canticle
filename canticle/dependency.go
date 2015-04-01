package canticle

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
)

// A CanticleDependency defines deps for this package to pull
type Dependency struct {
	ImportPath string
	SourcePath string `json:",omitempty"`
	Revision   string `json:",omitempty"`
	Comment    string `json:",omitempty"`
}

type Dependencies map[string]*Dependency

func NewDependencies() Dependencies {
	return make(map[string]*Dependency)
}

// Dependency returns nil if importPath is not present
func (c Dependencies) Dependency(importPath string) *Dependency {
	return c[importPath]
}

// AddDependency adds a dependency if its import path is not already
// present.  If the import path is present and the revision or source
// path are not present the following merge strategy will be taken:
// *  A non blank ("") revision will always be prefered to a blank one
// *  A non blank ("") source path will always be prefered to a blank one
// *  If their is a conflicting revision the current one will be taken
// *  If their is a conflicting sourcepath the current one will be taken
func (c Dependencies) AddDependency(dep *Dependency) error {
	if _, found := c[dep.ImportPath]; !found {
		c[dep.ImportPath] = dep
		return nil
	}

	already := c[dep.ImportPath]
	var err error
	switch {
	case already.Revision == "" && dep.Revision != "":
		c[dep.ImportPath].Revision = dep.Revision
		err = fmt.Errorf("Dep %s was previously defined with no revision using %s\n", already.ImportPath, dep.Revision)
	case already.SourcePath == "" && dep.SourcePath != "":
		c[dep.ImportPath].SourcePath = dep.SourcePath
		err = fmt.Errorf("Dep %s was previously defined with no source path using %s\n", already.ImportPath, dep.SourcePath)
	case already.Revision != dep.Revision:
		err = fmt.Errorf("Dep %s has conflicting revisions %s and %s, using first\n", already.ImportPath, already.Revision, dep.Revision)
	case already.SourcePath != dep.SourcePath:
		err = fmt.Errorf("Dep %s has conflicting source %s and %s, using first", already.ImportPath, already.SourcePath, dep.SourcePath)
	}

	return err
}

// DepReaderr works in a particular gopath to read the
// dependencies of both Canticle and non-Canticle go packages.
type DepReader struct {
}

// ReadDependencies returns the dependencies listed in the
// packages Canticle file. Dependencies will never be nil.
func (dr *DepReader) ReadCanticleDependencies(p, gopath string) (Dependencies, error) {
	// If this package does not have canticle deps just load it
	c, err := ioutil.ReadFile(path.Join(gopath, "src", p, "Canticle"))
	if err != nil {
		return NewDependencies(), err
	}

	var d []*Dependency
	if err := json.Unmarshal(c, &d); err != nil {
		return NewDependencies(), err
	}

	deps := NewDependencies()
	for _, dep := range d {
		deps[dep.ImportPath] = dep
	}

	return deps, nil
}

// ReadRemoteDependencies reads the dependencies for package p listed
// as imports in *.go files, including tests, and returns the result.
func (dr *DepReader) ReadRemoteDependencies(p, gopath string) ([]string, error) {
	pkg, err := LoadPackage(p, gopath)
	if err != nil {
		return []string{}, err
	}
	return pkg.RemoteImports(true), nil
}

// ReadDependencies will return both the Canticle (if present)
// dependencies for a package p and the remote dependencies.
func (dr *DepReader) ReadDependencies(p, gopath string) (Dependencies, error) {
	deps, err := dr.ReadCanticleDependencies(p, gopath)
	if err != nil && !os.IsNotExist(err) {
		return deps, err
	}

	// Load any non Canticle deps
	remotes, err := dr.ReadRemoteDependencies(p, gopath)
	if err != nil {
		return deps, err
	}

	for _, remote := range remotes {
		deps.AddDependency(&Dependency{ImportPath: remote})
	}

	return deps, nil
}

type DependencyReader interface {
	ReadDependencies(p, gopath string) (Dependencies, error)
}

type RepoResolver interface {
	ResolveRepo(importPath, url string) (VCS, error)
}

// DependencyLoader is used to load and resolve the
// dependencies of a package.  It will load the dependencies for an
// import path only once. After that it uses a memoized copy.
type DependencyLoader struct {
	// Memoization of loaded packages
	loadedPackages map[string]bool
	deps           Dependencies

	depReader DependencyReader
	resolver  RepoResolver
	gopath    string
	// If Update is true repos will be fetched using the
	// supplied repoResolver and/or set to the revision specified
	// in by the dependencyReader
	Update bool
	// If IgnoreErrors is true repo resolution will continue even
	// if some repos failures are hit.
	IgnoreErrors bool
}

// NewDependencyLoader creates a new dep loader. It uses the
// specified resolver and depReader to load dependencies. Both must
// not be nil.
func NewDependencyLoader(resolver RepoResolver, depReader DependencyReader, gopath string) *DependencyLoader {
	return &DependencyLoader{
		loadedPackages: make(map[string]bool),
		deps:           NewDependencies(),
		depReader:      depReader,
		resolver:       resolver,
		gopath:         gopath,
	}
}

// SaveDependencies
func (dl *DependencyLoader) SaveDependencies() Dependencies {
	return nil
}

// LoadAllPackageDependencies is equivalent to
// LoadAllDependencies(&Dependency{ImportPath: p})
func (dl *DependencyLoader) LoadAllPackageDependencies(p string) (Dependencies, error) {
	return dl.LoadAllDependencies(&Dependency{ImportPath: p})
}

// LoadAllDependencies recursively reads and loads all dependencies of
// dep and returns the result. If dl.Update is true it will fetch non
// existent repos and/or set them to the versions returned by teh
// depReader.
func (dl *DependencyLoader) LoadAllDependencies(dep *Dependency) (Dependencies, error) {
	// If we have already loaded this package do not load it again
	if loaded := dl.loadedPackages[dep.ImportPath]; loaded {
		return dl.deps, nil
	}

	// If this package isn't on disk fetch it
	if dl.Update {
		if err := dl.FetchUpdatePackage(dep); err != nil {
			return nil, err
		}
	}

	// Load this package
	deps, err := dl.depReader.ReadDependencies(dep.ImportPath, dl.gopath)
	if err != nil {
		fmt.Println("Error Loading Package: ", err.Error())
		return nil, err
	}

	// Merge its deps
	for _, dep := range deps {
		dl.deps.AddDependency(dep)
	}

	dl.loadedPackages[dep.ImportPath] = true

	// Load the child packages
	for _, dep := range deps {
		if dl.loadedPackages[dep.ImportPath] {
			continue
		}
		if _, err := dl.LoadAllDependencies(dep); err != nil {
			return nil, err
		}
	}

	return dl.deps, nil
}

// FetchUpdatePackage will fetch or set the specified package to the version
// defined by the Dependency or if no version is defined will use
// the VCS default.
func (dl *DependencyLoader) FetchUpdatePackage(dep *Dependency) error {
	s, err := os.Stat(path.Join(dl.gopath, "src", dep.ImportPath))
	switch {
	case os.IsPermission(err):
		return fmt.Errorf("Package %s exists but could not be accessed (permissions)", dep.ImportPath)
	case s != nil && !s.IsDir():
		return fmt.Errorf("Package %s is a file not a directory", dep.ImportPath)
	case os.IsNotExist(err):
		fmt.Printf("Fetching package: %+v\n", dep)
		return dl.FetchRepo(dep)
	default:
		fmt.Printf("Package %+v already exists\n", dep)
		return dl.UpdateRepo(dep)
	}
}

// UpdateRepo sets the repo to the correct version as defined by the
// Dependency.
func (dl *DependencyLoader) UpdateRepo(dep *Dependency) error {
	if dep.Revision == "" {
		return nil
	}

	vcs, err := dl.resolver.ResolveRepo(dep.ImportPath, dep.SourcePath)
	if err != nil {
		fmt.Println("Error resolving repo: ", err.Error())
		return err
	}

	return vcs.SetRev(dep.Revision)
}

// FetchRepo fetches a non created repo at the version defined by the
// Dependency. If no version is defined the default checkout is
// used as defined by the vcs.
func (dl *DependencyLoader) FetchRepo(dep *Dependency) error {
	vcs, err := dl.resolver.ResolveRepo(dep.ImportPath, dep.SourcePath)
	if err != nil {
		fmt.Println("Error resolving repo: ", err.Error())
		return err
	}

	if err := vcs.Create(dep.Revision); err != nil {
		fmt.Println("Error at CreateVCS: ", err.Error())
		return err
	}

	return nil
}
