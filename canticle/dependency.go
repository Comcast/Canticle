package canticle

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
)

// A Dependency defines all information about this package.
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
// present. If the import path is present and the revision or source
// path are not present the following merge strategy will be taken:
// *  A non blank ("") revision will always be prefered to a blank one
// *  A non blank ("") source path will always be prefered to a blank one
// *  If there is a conflicting revision the current one will be taken,
// an error will be returned
// *  If there is a conflicting sourcepath the current one will be taken,
// an error will be returned
// *  Comments are never merge
func (c Dependencies) AddDependency(dep *Dependency) []error {
	if _, found := c[dep.ImportPath]; !found {
		c[dep.ImportPath] = dep
		return nil
	}

	already := c[dep.ImportPath]
	var errs []error
	switch {
	case already.Revision == "" && dep.Revision != "":
		c[dep.ImportPath].Revision = dep.Revision
	case already.SourcePath == "" && dep.SourcePath != "":
		c[dep.ImportPath].SourcePath = dep.SourcePath
	case already.Revision != dep.Revision:
		errs = append(errs, fmt.Errorf("Dep %s has conflicting revisions %s and %s, using first\n", already.ImportPath, already.Revision, dep.Revision))
	case already.SourcePath != dep.SourcePath:
		errs = append(errs, fmt.Errorf("Dep %s has conflicting source %s and %s, using first", already.ImportPath, already.SourcePath, dep.SourcePath))
	}

	return errs
}

// DepReaderr works in a particular gopath to read the
// dependencies of both Canticle and non-Canticle go packages.
type DepReader struct {
	Gopath string
}

// ReadDependencies returns the dependencies listed in the
// packages Canticle file. Dependencies will never be nil.
func (dr *DepReader) ReadCanticleDependencies(p string) (Dependencies, error) {
	// If this package does not have canticle deps just load it
	c, err := ioutil.ReadFile(path.Join(dr.Gopath, "src", p, "Canticle"))
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
func (dr *DepReader) ReadRemoteDependencies(p string) ([]string, error) {
	pkg, err := LoadPackage(p, dr.Gopath)
	if err != nil {
		return []string{}, err
	}
	return pkg.RemoteImports(true), nil
}

// ReadDependencies will return both the Canticle (if present)
// dependencies for a package p and the remote dependencies.
func (dr *DepReader) ReadDependencies(p string) (Dependencies, error) {
	deps, err := dr.ReadCanticleDependencies(p)
	if err != nil && !os.IsNotExist(err) {
		return deps, err
	}

	// Load any non Canticle deps
	remotes, err := dr.ReadRemoteDependencies(p)
	if err != nil {
		return deps, err
	}

	for _, remote := range remotes {
		dep := &Dependency{ImportPath: remote}
		if deps.Dependency(dep.ImportPath) == nil {
			deps.AddDependency(dep)
		}
	}

	return deps, nil
}

// PkgReaderFunc takes a given package string and resolves
// the dependencies for that package. If error is not nil on
// return the walker halts and returns the error.
type PkgReaderFunc func(pkg string) (Dependencies, error)

// DepHandlerFunc is called once for each loaded package. If the error
// ErrorSkip is returned deps or this package are no read. All other
// non nil errors halt the walker and return the value. Errs will contain
// any errors detected when loading a dependency.
type DepHandlerFunc func(dep *Dependency, errs []error) error

// ErrorSkip tells a walker to skip loading the deps of this dep.
var ErrorSkip = errors.New("Skip this dep")

// DependencyWalker is used to walker the dependencies of a package.
// It will walk the dependencies for an import path only once.
type DependencyWalker struct {
	// Memoization of loaded packages
	depsQueue      []*Dependency
	loadedPackages Dependencies
	readPackage    PkgReaderFunc
	handleDep      DepHandlerFunc
}

// NewDependencyWalker creates a new dep loader. It uses the
// specified  depReader to load dependencies. It will call the handler
// with the resulting dependencies.
func NewDependencyWalker(reader PkgReaderFunc, handler DepHandlerFunc) *DependencyWalker {
	return &DependencyWalker{
		loadedPackages: NewDependencies(),
		handleDep:      handler,
		readPackage:    reader,
	}
}

// TraversePackageDependencies is equivalent too:
// dw.TraverseDependencies(&Dependency{ImportPath: p})
func (dw *DependencyWalker) TraversePackageDependencies(p string) error {
	return dw.TraverseDependencies(&Dependency{ImportPath: p})
}

// TraverseDependencies reads and loads all dependencies of dep. It is
// a breadth first search. If handler returns the special error
// ErrorSkip it does not read the deps of this package.
func (dw *DependencyWalker) TraverseDependencies(dep *Dependency) error {
	dw.depsQueue = append(dw.depsQueue, dep)
	for len(dw.depsQueue) > 0 {
		// Dequeue and mark loaded
		dep = dw.depsQueue[0]
		dw.depsQueue = dw.depsQueue[1:]
		errs := dw.loadedPackages.AddDependency(dep)

		// Inform our handler of this package
		err := dw.handleDep(dep, errs)
		switch {
		case err == ErrorSkip:
			continue
		case err != nil:
			return err
		}
		// Load this package
		deps, err := dw.readPackage(dep.ImportPath)
		if err != nil {
			return err
		}
		// Load the child packages
		for _, dep := range deps {
			// If we already traversed this node don't re-queue it
			if dw.loadedPackages.Dependency(dep.ImportPath) != nil {
				continue
			}
			// Push back the dep into the queue
			dw.depsQueue = append(dw.depsQueue, dep)
		}
	}

	return nil
}

// A ResolverFunc transforms an importpath and source for a
// dependency.  to a VCS which can be used to modify and get
// information about that repo.
type ResolverFunc func(importPath, source string) (VCS, error)

// A DependencyLoader fetches and set the correct revision for a
// dependency using the specified resolver.
type DependencyLoader struct {
	deps    Dependencies
	gopath  string
	resolve ResolverFunc
	// If HaltOnError is true the DependencyLoader
	// will stop the fetching process if a dependency
	// conflict is detected. Defaults to true.
	HaltOnError bool
}

// NewDependencyLoader returns a DependencyLoader initialized with the
// resolver func.
func NewDependencyLoader(resolver ResolverFunc, gopath string) *DependencyLoader {
	return &DependencyLoader{
		deps:        NewDependencies(),
		resolve:     resolver,
		gopath:      gopath,
		HaltOnError: true,
	}
}

// FetchUpdatePackage will fetch or set the specified package to the version
// defined by the Dependency or if no version is defined will use
// the VCS default.
func (dl *DependencyLoader) FetchUpdatePackage(dep *Dependency, errs []error) error {
	if len(errs) > 0 {
		for _, err := range errs {
			log.Printf("Errors loading dependency: %s", err.Error())
		}
		if dl.HaltOnError {
			return errors.New("Package fetcher halting on error")
		}
	}

	// If we already fetched the package continue
	if dl.deps.Dependency(dep.ImportPath) != nil {
		return nil
	}
	// Else attempt to resolve and fetch the package
	dl.deps.AddDependency(dep)
	vcs, err := dl.resolve(dep.ImportPath, dep.SourcePath)
	if err != nil {
		return err
	}

	s, err := os.Stat(path.Join(dl.gopath, "src", dep.ImportPath))
	switch {
	case os.IsPermission(err):
		return fmt.Errorf("Package %s exists but could not be accessed (permissions)", dep.ImportPath)
	case s != nil && !s.IsDir():
		return fmt.Errorf("Package %s is a file not a directory", dep.ImportPath)
	case os.IsNotExist(err):
		return vcs.Create(dep.Revision)
	default:
		if dep.Revision == "" {
			return nil
		}
		return vcs.SetRev(dep.Revision)
	}
}

// FetchedDeps returns a copy of the dependencies fetched using
// fetchupdatepackage.
func (dl *DependencyLoader) FetchedDeps() Dependencies {
	return dl.deps
}
