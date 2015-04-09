package canticle

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"sort"
)

// A Dependency defines all information about this package.
type Dependency struct {
	ImportPath string
	SourcePath string `json:",omitempty"`
	Revision   string `json:",omitempty"`
	Comment    string `json:",omitempty"`
}

// Dependencies is the set of Dependencies for a package. This is
// stored as a map from ImportPath to Dependency.
type Dependencies map[string]*Dependency

// NewDependencies creates a new empty dependencies structure/
func NewDependencies() Dependencies {
	return make(map[string]*Dependency)
}

// Dependency returns nil if importPath is not present.
func (c Dependencies) Dependency(importPath string) *Dependency {
	return c[importPath]
}

// String will print this out as newline seperated %+v values.
func (c Dependencies) String() string {
	str := ""
	for _, dep := range c {
		str += fmt.Sprintf("%+v\n", dep)
	}
	return str
}

// MarshalJSON will encode this as a JSON array.
func (c Dependencies) MarshalJSON() ([]byte, error) {
	deps := make([]*Dependency, 0, len(c))
	for _, dep := range c {
		deps = append(deps, dep)
	}
	return json.Marshal(deps)
}

// UnmarshalJSON will decode this from a JSON array.
func (c Dependencies) UnmarshalJSON(data []byte) error {
	var deps []*Dependency
	if err := json.Unmarshal(data, &deps); err != nil {
		return err
	}
	for _, dep := range deps {
		c.AddDependency(dep)
	}
	return nil
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

// RemoveDependency will delete a dep based on its importpath from
// this dep tree.
func (c Dependencies) RemoveDependency(dep *Dependency) {
	delete(c, dep.ImportPath)
}

// DepReader works in a particular gopath to read the
// dependencies of both Canticle and non-Canticle go packages.
type DepReader struct {
	Gopath string
}

// ReadCanticleDependencies returns the dependencies listed in the
// packages Canticle file. Dependencies will never be nil.
func (dr *DepReader) ReadCanticleDependencies(p string) (Dependencies, error) {
	deps := NewDependencies()
	f, err := os.Open(path.Join(PackageSource(dr.Gopath, p), "Canticle"))
	if err != nil {
		return deps, err
	}

	d := json.NewDecoder(f)
	if err := d.Decode(&deps); err != nil {
		return deps, err
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
		children, err := dw.readPackage(dep.ImportPath)
		if err != nil {
			return err
		}

		// Queue the child packages
		// Make sure we iterate over them in a stable manner
		childKeys := make([]string, 0, len(children))
		for k := range children {
			childKeys = append(childKeys, k)
		}
		sort.Strings(childKeys)

		for _, k := range childKeys {
			child := children.Dependency(k)
			LogVerbose("Package %+v has dep %+v", dep, child)
			// If we already traversed this node don't re-queue it
			if dw.loadedPackages.Dependency(child.ImportPath) != nil {
				continue
			}
			// Push back the dep into the queue
			dw.depsQueue = append(dw.depsQueue, child)
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
	s, err := os.Stat(path.Join(dl.gopath, "src", dep.ImportPath))
	switch {
	case s != nil && !s.IsDir():
		return fmt.Errorf("Package %s is a file not a directory", dep.ImportPath)
	case os.IsNotExist(err):
		LogVerbose("Create package %s at revision %s", dep.ImportPath, dep.Revision)
		vcs, err := dl.resolve(dep.ImportPath, dep.SourcePath)
		if err != nil {
			return err
		}
		return vcs.Create(dep.Revision)
	case s != nil && s.IsDir() && dep.Revision == "":
		LogVerbose("Package already on disk %s", dep.ImportPath)
		return nil
	default:
		LogVerbose("Setting package %s to revision %s", dep.ImportPath, dep.Revision)
		vcs, err := dl.resolve(dep.ImportPath, dep.SourcePath)
		if err != nil {
			return err
		}
		return vcs.SetRev(dep.Revision)
	}
}

// FetchedDeps returns a copy of the dependencies fetched using
// fetchupdatepackage.
func (dl *DependencyLoader) FetchedDeps() Dependencies {
	return dl.deps
}

// DependencySaver is a handler for dependencies that will save all
// dependencies current revisions. Call Dependencies() to retrieve the
// loaded Dependencies.
type DependencySaver struct {
	deps    Dependencies
	gopath  string
	resolve ResolverFunc
}

// NewDependencySaver builds a new dependencysaver to work in the
// specified gopath and resolve using the resolverfunc. A
// DependencySaver should generally only be used once. A
// DependencySaver will not attempt to load remote dependencies even
// if the resolverfunc can handle them.
func NewDependencySaver(resolver ResolverFunc, gopath string) *DependencySaver {
	return &DependencySaver{
		deps:    NewDependencies(),
		resolve: resolver,
		gopath:  gopath,
	}
}

// SavePackageRevision will attempt to load the revision of the
// package dep and add it to our Dependencies(). If errs contains any
// errors it will halt. If the package can not be loaded from gopath
// (is not a directory, does not exist, etc.) an error will be
// returned.
func (ds *DependencySaver) SavePackageRevision(dep *Dependency, errs []error) error {
	for _, err := range errs {
		log.Printf("Errors loading dependency: %s", err.Error())
	}
	if len(errs) > 0 {
		return errors.New("Package saver halting on error")
	}

	s, err := os.Stat(PackageSource(ds.gopath, dep.ImportPath))
	switch {
	case s != nil && !s.IsDir():
		return fmt.Errorf("Package %s is a file not a directory", dep.ImportPath)
	case os.IsNotExist(err):
		return fmt.Errorf("Package %s could not be found on disk", dep.ImportPath)
	case err != nil:
		return err
	}
	vcs, err := ds.resolve(dep.ImportPath, dep.SourcePath)
	if err != nil {
		return err
	}
	dep.Revision, err = vcs.GetRev()
	if err != nil {
		return err
	}
	dep.SourcePath, err = vcs.GetSource()
	if err != nil {
		return err
	}
	ds.deps.AddDependency(dep)
	return nil
}

// Dependencies returns the resolved dependencies from dependency
// saver.
func (ds *DependencySaver) Dependencies() Dependencies {
	return ds.deps
}
