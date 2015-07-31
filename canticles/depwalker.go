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
type DepReaderFunc func(importPath string) (Dependencies, error)

// A DependencyLoader fetches and set the correct revision for a
// dependency using the specified resolver.
type DependencyLoader struct {
	deps     Dependencies
	gopath   string
	resolver RepoResolver
	read     DepReaderFunc
}

// NewDependencyLoader returns a DependencyLoader initialized with the
// resolver func.
func NewDependencyLoader(resolver RepoResolver, reader DepReaderFunc, gopath string) *DependencyLoader {
	return &DependencyLoader{
		deps:     NewDependencies(),
		read:     reader,
		resolver: resolver,
		gopath:   gopath,
	}
}

// FetchUpdatePackage will fetch or set the specified package to the version
// defined by the Dependency or if no version is defined will use
// the VCS default.
func (dl *DependencyLoader) FetchUpdatePackage(pkg string) error {
	LogVerbose("DepLoader handling pkg: %s", pkg)

	// See if this import path is on disk
	fetch := false
	s, err := os.Stat(PackageSource(dl.gopath, pkg))
	switch {
	case err != nil && os.IsNotExist(err):
		fetch = true
	case err != nil:
		fmt.Errorf("cant fetch package error when stating import path %s", err.Error())
	case s != nil && !s.IsDir():
		return fmt.Errorf("cant fetch package %s is a file not a directory", pkg)
	}

	// Fetch or update the package
	dep := dl.deps.Dependency(pkg)
	if dep == nil {
		dep = NewDependency(pkg)
		LogVerbose("Creating Dep: %+v", dep)
	}

	// Resolve the vcs
	vcs, err := dl.resolver.ResolveRepo(pkg, nil)
	if err != nil {
		return fmt.Errorf("resolving package %s version control %s", pkg, err.Error())
	}

	// Fetch or set
	if fetch {
		err = dl.fetchPackage(vcs, dep)
	} else {
		err = dl.setRevision(vcs, dep)
	}
	if err != nil {
		return fmt.Errorf("cant fetch package %s %s", pkg, err.Error())
	}

	// Load the canticle deps file of our package and save it, not
	// having the file is not an error.
	deps, err := dl.read(pkg)
	switch {
	case err != nil && !os.IsNotExist(err):
		return fmt.Errorf("cant fetch package %s couldn't read cant file %s", pkg, err.Error())
	case err != nil && os.IsNotExist(err):
		return nil
	}
	LogVerbose("Read package %s deps:\n[\n%+v]", pkg, deps)
	dl.deps.AddDependencies(deps)
	return nil
}

func (dl *DependencyLoader) setRevision(vcs VCS, dep *Dependency) error {
	LogVerbose("Setting rev on dep %+v", dep)
	if err := vcs.SetRev(""); err != nil {
		return fmt.Errorf("failed to set revision because %s", err.Error())
	}
	return nil
}

func (dl *DependencyLoader) fetchPackage(vcs VCS, dep *Dependency) error {
	LogVerbose("Fetching dep %+v", dep)
	if err := vcs.Create(""); err != nil {
		return fmt.Errorf("failed to fetch because %s", err.Error())
	}
	return nil
}

// FetchedDeps returns a copy of the dependencies fetched using
// fetchupdatepackage.
func (dl *DependencyLoader) FetchedDeps() Dependencies {
	return dl.deps
}

/*
// ProjectSaver is a handler for dependencies that will save all
// dependencies current revisions. Call Dependencies() to retrieve the
// loaded Dependencies.
type ProjectSaver struct {
	deps     Dependencies
	resolver RepoResolver
	rootPath string
}

// NewProjectSaver uses the VCS repo resolver and the rootPath to resolved
// any vcs systems below rootpath.
func NewProjectSaver(resolver RepoResolver, rootPath string) *ProjectSaver {
	return &ProjectSaver{
		deps:     NewDependencies(),
		resolver: resolver,
		rootPath: rootPath,
	}
}

// SaveProjectPath adds this path to the project as a dep if the vcs
// root is not a parent of or equal to the rootPath for the project.
// E.g. don't save ourselves as a dep of ourselves cause thats stupid.
func (ps *ProjectSaver) SaveProjectPath(path string) error {
	// Resolve any vcs repo, if  we can't find it thats fine read the children
	vcs, err := ps.resolver.ResolveRepo(path, nil)
	if err != nil {
		LogVerbose("No vcs found at path %s %s", path, err.Error())
		return nil
	}
	dep := &Dependency{}
	dep.Root = vcs.GetRoot()
	// If its VCS is part of the project ignore it and traverse down
	if dep.Root == ps.rootPath || PathIsChild(ps.rootPath, dep.Root) {
		return nil
	}
	rev, err := vcs.GetRev()
	if err != nil {
		return fmt.Errorf("cant get revision from vcs at %s %s", dep.Root, err.Error())
	}
	dep.Revisions.Add(rev)
	source, err := vcs.GetSource()
	if err != nil {
		return fmt.Errorf("cant get vcs source from vcs at %s %s", dep.Root, err.Error())
	}
	dep.Sources.Add(source)
	ps.deps.AddDependency(dep)
	return ErrorSkip
}

// Returns the resolved dependencies for this project. May be empty,
// will not be null.
func (ps *ProjectSaver) Dependencies() Dependencies {
	return ps.deps
}
*/
// DependencySaver is a handler for dependencies that will save all
// dependencies current revisions. Call Dependencies() to retrieve the
// loaded Dependencies.
type DependencySaver struct {
	deps   Dependencies
	gopath string
	read   DepReaderFunc
}

// NewDependencySaver builds a new dependencysaver to work in the
// specified gopath and resolve using the resolverfunc. A
// DependencySaver should generally only be used once. A
// DependencySaver will not attempt to load remote dependencies even
// if the resolverfunc can handle them. Deps that resolve using ignore
// will not be saved.
func NewDependencySaver(reader DepReaderFunc, gopath string) *DependencySaver {
	return &DependencySaver{
		deps:   NewDependencies(),
		read:   reader,
		gopath: gopath,
	}
}

// SavePackageDeps uses the reader to read all 1st order deps of this
// pkg.
func (ds *DependencySaver) SavePackageDeps(path string) error {
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
	pkg, err := PackageName(ds.gopath, path)
	if err != nil {
		return fmt.Errorf("Error getting package name for path %s", path)
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

// PackagePaths returns all subdirs and all import paths for a pkg.
func (ds *DependencySaver) PackagePaths(pkg string) ([]string, error) {
	subdirs, err := VisibleSubDirectories(pkg)
	if err != nil {
		return []string{}, err
	}
	LogVerbose("Package has subdirs %v", subdirs)
	dep := ds.deps.Dependency(pkg)
	if dep == nil {
		return subdirs, nil
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
