package canticles

import (
	"errors"
	"fmt"
	"os"
	"sort"
)

// PkgReaderFunc takes a given package string and returns all
// the dependencies for that package. If error is not nil on
// return the walker halts and returns the error.
type PkgReaderFunc func(pkg string) ([]string, error)

// PkgHandlerFunc is called once for each loaded package. If the error
// ErrorSkip is returned deps or this package are no read. All other
// non nil errors halt the walker and return the value.
type PkgHandlerFunc func(pkg string) error

// ErrorSkip tells a walker to skip loading the deps of this dep.
var ErrorSkip = errors.New("skip this dep")

// DependencyWalker is used to walker the dependencies of a package.
// It will walk the dependencies for an import path only once.
type DependencyWalker struct {
	nodeQueue   []string
	visited     map[string]bool
	readPackage PkgReaderFunc
	handleDep   PkgHandlerFunc
}

// NewDependencyWalker creates a new dep loader. It uses the
// specified  depReader to load dependencies. It will call the handler
// with the resulting dependencies.
func NewDependencyWalker(reader PkgReaderFunc, handler PkgHandlerFunc) *DependencyWalker {
	return &DependencyWalker{
		visited:     make(map[string]bool),
		handleDep:   handler,
		readPackage: reader,
	}
}

// TraverseDependencies reads and loads all dependencies of dep. It is
// a breadth first search. If handler returns the special error
// ErrorSkip it does not read the deps of this package.
func (dw *DependencyWalker) TraverseDependencies(pkg string) error {
	dw.nodeQueue = append(dw.nodeQueue, pkg)
	for len(dw.nodeQueue) > 0 {
		// Dequeue and mark loaded
		p := dw.nodeQueue[0]
		dw.nodeQueue = dw.nodeQueue[1:]
		dw.visited[p] = true
		LogVerbose("Handling pkg: %+v", p)

		// Inform our handler of this package
		err := dw.handleDep(p)
		switch {
		case err == ErrorSkip:
			continue
		case err != nil:
			return err
		}

		// Read out our children
		children, err := dw.readPackage(p)
		if err != nil {
			return fmt.Errorf("cant read deps of package %s with error %s", pkg, err.Error())
		}
		sort.Strings(children)
		LogVerbose("Package %s has children %v", p, children)

		for _, child := range children {
			if dw.visited[child] {
				continue
			}
			dw.nodeQueue = append(dw.nodeQueue, child)
		}
	}

	return nil
}

// A DepReader reads the set of deps for a package
type DependencyReader interface {
	CanticleDependencies(importPath string) ([]*CanticleDependency, error)
	AllDeps(importPath string) (Dependencies, error)
}

// A DependencyLoader fetches and set the correct revision for a
// dependency using the specified resolver.
type DependencyLoader struct {
	deps      Dependencies
	cdeps     map[string]*CanticleDependency
	gopath    string
	root      string
	resolver  RepoResolver
	depReader DependencyReader
}

// NewDependencyLoader returns a DependencyLoader initialized with the
// resolver func.
func NewDependencyLoader(resolver RepoResolver, depReader DependencyReader, gopath, root string) *DependencyLoader {
	return &DependencyLoader{
		deps:      NewDependencies(),
		cdeps:     make(map[string]*CanticleDependency),
		depReader: depReader,
		resolver:  resolver,
		gopath:    gopath,
		root:      root,
	}
}

// FetchUpdatePackage will fetch or set the specified path to the version
// defined by the Dependency or if no version is defined will use
// the VCS default.
func (dl *DependencyLoader) FetchUpdatePath(path string) error {
	LogVerbose("DepLoader handling path: %s", path)

	// See if this path is on disk
	fetch := false
	s, err := os.Stat(path)
	switch {
	case err != nil && os.IsNotExist(err):
		fetch = true
	case err != nil:
		fmt.Errorf("cant fetch package error when stating import path %s", err.Error())
	case s != nil && !s.IsDir():
		return fmt.Errorf("cant fetch pkg for path %s is a file not a directory", path)
	}

	// Get the packagename
	pkg, err := PackageName(dl.gopath, path)
	if err != nil {
		return err
	}

	// Fetch or update the package
	// Resolve the vcs
	cdep := dl.cdeps[pkg]
	vcs, err := dl.resolver.ResolveRepo(pkg, cdep)
	if err != nil {
		return fmt.Errorf("resolving package %s version control %s", pkg, err.Error())
	}

	// If this is a new dep (from a non saved source)
	// fill out its details
	if cdep == nil {
		sauce, err := vcs.GetSource()
		if err != nil {
			return err
		}

		cdep = &CanticleDependency{
			Root:       vcs.GetRoot(),
			SourcePath: sauce,
		}
		dl.cdeps[pkg] = cdep
	}

	// Fetch or set
	if fetch {
		err = dl.fetchPackage(vcs, cdep)
	} else {
		err = dl.setRevision(vcs, cdep)
	}
	if err != nil {
		return fmt.Errorf("cant fetch package %s %s", pkg, err.Error())
	}

	// Load the canticle deps file of our package and save it, not
	// having the file is not an error.
	cdeps, err := dl.depReader.CanticleDependencies(pkg)
	switch {
	case err != nil && !os.IsNotExist(err):
		return fmt.Errorf("cant fetch package %s couldn't read cant file %s", pkg, err.Error())
	case err != nil && os.IsNotExist(err):
		return nil
	}
	LogVerbose("Read package canticle %s deps:\n[\n%+v]", pkg, cdeps)
	for _, pkgDep := range cdeps {
		if dl.cdeps[pkgDep.Root] == nil {
			dl.cdeps[pkgDep.Root] = pkgDep
		}
	}

	// Load all the dep for this file directly
	deps, err := dl.depReader.AllDeps(pkg)
	if err != nil {
		return fmt.Errorf("cant fetch package %s couldn't read deps %s", pkg, err.Error())
	}
	LogVerbose("Read package %s deps:\n[\n%+v]", pkg, deps)
	dl.deps.AddDependencies(deps)

	return nil
}

//
func (dl *DependencyLoader) PackagePaths(path string) ([]string, error) {
	// The set of packagepaths for a pkg is determined by:
	// Its imports from dl.readDeps.
	// A cdep with All specified as true.
	//   iif a cdep with all is specified a different
	//   reader that recurs subfolders should be used
	var subdirs []string
	var err error
	if PathIsChild(dl.root, path) {
		subdirs, err = VisibleSubDirectories(path)
		if err != nil {
			return []string{}, err
		}
		LogVerbose("Package has subdirs %v", subdirs)
	}

	pkg, err := PackageName(dl.gopath, path)
	if err != nil {
		return []string{}, err
	}

	deps := dl.deps.Dependency(pkg)
	for root, cdep := range dl.cdeps {
		if cdep.All {
			deps.Imports.Add(root)
		}
	}
	return append(deps.Imports.Array(), subdirs...), nil
}

func (dl *DependencyLoader) setRevision(vcs VCS, dep *CanticleDependency) error {
	LogVerbose("Setting rev on dep %+v", dep)
	if err := vcs.SetRev(""); err != nil {
		return fmt.Errorf("failed to set revision because %s", err.Error())
	}
	return nil
}

func (dl *DependencyLoader) fetchPackage(vcs VCS, dep *CanticleDependency) error {
	LogVerbose("Fetching dep %+v", dep)
	if err := vcs.Create(""); err != nil {
		return fmt.Errorf("failed to fetch because %s", err.Error())
	}
	return nil
}

type DepReaderFunc func(importPath string) (Dependencies, error)

// DependencySaver is a handler for dependencies that will save all
// dependencies current revisions. Call Dependencies() to retrieve the
// loaded Dependencies.
type DependencySaver struct {
	deps   Dependencies
	gopath string
	root   string
	read   DepReaderFunc
}

// NewDependencySaver builds a new dependencysaver to work in the
// specified gopath and resolve using the resolverfunc. A
// DependencySaver should generally only be used once. A
// DependencySaver will not attempt to load remote dependencies even
// if the resolverfunc can handle them. Deps that resolve using ignore
// will not be saved.
func NewDependencySaver(reader DepReaderFunc, gopath, root string) *DependencySaver {
	return &DependencySaver{
		deps:   NewDependencies(),
		root:   root,
		read:   reader,
		gopath: gopath,
	}
}

// SavePackageDeps uses the reader to read all 1st order deps of this
// pkg.
func (ds *DependencySaver) SavePackageDeps(path string) error {
	pkg, err := PackageName(ds.gopath, path)
	if err != nil {
		return fmt.Errorf("Error getting package name for path %s", path)
	}

	// Check if we can find this package
	s, err := os.Stat(path)
	switch {
	case s != nil && !s.IsDir():
		err = fmt.Errorf("cant save deps for path %s is a file not a directory", path)
	case err != nil && os.IsNotExist(err):
		err = fmt.Errorf("cant save deps for path %s could not be found on disk", path)
	case err != nil:
		err = fmt.Errorf("cant save deps for path %s due to %s", path, err.Error())
	}
	if err != nil {
		LogVerbose("Error stating path %s %s", path, err.Error())
		dep := NewDependency(pkg)
		dep.Err = err
		ds.deps.AddDependency(dep)
		return ErrorSkip
	}

	pkgDeps, err := ds.read(path)
	if err != nil {
		LogVerbose("Error reading pkg deps %s %s", pkg, err.Error())
		dep := NewDependency(pkg)
		dep.Err = fmt.Errorf("cant read deps for package %s %s", pkg, err.Error())
		ds.deps.AddDependency(dep)
		return nil
	}
	if len(pkgDeps) == 0 {
		return nil
	}

	dep := NewDependency(pkg)
	for _, d := range pkgDeps {
		d.ImportedFrom.Add(pkg)
	}
	ds.deps.AddDependencies(pkgDeps)
	for _, pkgDep := range pkgDeps {
		dep.Imports.Add(pkgDep.ImportPath)
	}
	ds.deps.AddDependency(dep)
	return nil
}

// PackagePaths returns d all import paths for a pkg, and all subdirs
// if the pkg is under the root of the passed to the ds at construction.
func (ds *DependencySaver) PackagePaths(pkg string) ([]string, error) {
	var subdirs []string
	var err error
	if PathIsChild(ds.root, pkg) {
		subdirs, err = VisibleSubDirectories(pkg)
		if err != nil {
			return []string{}, err
		}
		LogVerbose("Package has subdirs %v", subdirs)
	}
	dep := ds.deps.Dependency(pkg)
	if dep == nil {
		return subdirs, nil
	}
	if dep.Err != nil {
		return []string{}, nil
	}
	imports := dep.Imports.Array()
	for i, imp := range imports {
		imports[i] = PackageSource(ds.gopath, imp)
	}
	LogVerbose("Package has imports %v", imports)
	return append(subdirs, imports...), nil
}

// Dependencies returns the resolved dependencies from dependency
// saver.
func (ds *DependencySaver) Dependencies() Dependencies {
	return ds.deps
}
