package canticle

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
var ErrorSkip = errors.New("Skip this dep")

// DependencyWalker is used to walker the dependencies of a package.
// It will walk the dependencies for an import path only once.
type DependencyWalker struct {
	// Memoization of loaded packages
	pkgQueue       []string
	loadedPackages map[string]bool
	readPackage    PkgReaderFunc
	handleDep      PkgHandlerFunc
}

// NewDependencyWalker creates a new dep loader. It uses the
// specified  depReader to load dependencies. It will call the handler
// with the resulting dependencies.
func NewDependencyWalker(reader PkgReaderFunc, handler PkgHandlerFunc) *DependencyWalker {
	return &DependencyWalker{
		loadedPackages: make(map[string]bool),
		handleDep:      handler,
		readPackage:    reader,
	}
}

// TraverseDependencies reads and loads all dependencies of dep. It is
// a breadth first search. If handler returns the special error
// ErrorSkip it does not read the deps of this package.
func (dw *DependencyWalker) TraverseDependencies(pkg string) error {
	dw.pkgQueue = append(dw.pkgQueue, pkg)
	for len(dw.pkgQueue) > 0 {
		// Dequeue and mark loaded
		p := dw.pkgQueue[0]
		dw.pkgQueue = dw.pkgQueue[1:]
		dw.loadedPackages[p] = true
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
			return err
		}

		sort.Strings(children)
		for _, child := range children {
			LogVerbose("Package %+v has dep %+v", p, child)
			// If we already traversed this node don't re-queue it
			if dw.loadedPackages[child] {
				continue
			}
			// Push back the dep into the queue
			dw.pkgQueue = append(dw.pkgQueue, child)
		}
	}

	return nil
}

// A canticle depreader reads the Canticle deps of a package
type CantDepReader func(importPath string) (Dependencies, error)

// A DependencyLoader fetches and set the correct revision for a
// dependency using the specified resolver.
type DependencyLoader struct {
	deps     Dependencies
	gopath   string
	resolver RepoResolver
	read     CantDepReader
}

// NewDependencyLoader returns a DependencyLoader initialized with the
// resolver func.
func NewDependencyLoader(resolver RepoResolver, reader CantDepReader, gopath string) *DependencyLoader {
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
		LogVerbose("Fetching package %s", pkg)
		fetch = true
	case err != nil:
		LogVerbose("Error stating import path %s", err.Error())
		return err
	case s != nil && !s.IsDir():
		return fmt.Errorf("Package %s is a file not a directory", pkg)
	}

	// Fetch or update the package
	dep := dl.deps.DepForImportPath(pkg)
	if dep == nil {
		dep = &Dependency{ImportPaths: []string{pkg}}
	}
	if fetch {
		dl.fetchPackage(pkg, dep)
	} else {
		dl.setRevision(pkg, dep)
	}

	// Load the canticle deps file of our package and save it, not
	// having the file is not an error.
	deps, err := dl.read(pkg)
	switch {
	case err != nil && !os.IsNotExist(err):
		return err
	case err != nil && os.IsNotExist(err):
		return nil
	}
	return dl.deps.AddDependencies(deps)
}

func (dl *DependencyLoader) setRevision(pkg string, dep *Dependency) error {
	vcs, err := dl.resolver.ResolveRepo(pkg, dep)
	if err != nil {
		LogVerbose("Failed to resolve package %s", pkg)
		return err
	}
	if err = vcs.SetRev(dep.Revision); err != nil {
		LogVerbose("Failed to create package %s", pkg)
		return err
	}
	return nil
}

func (dl *DependencyLoader) fetchPackage(pkg string, dep *Dependency) error {
	vcs, err := dl.resolver.ResolveRepo(pkg, dep)
	if err != nil {
		LogVerbose("Failed to resolve package %s", pkg)
		return err
	}
	if err = vcs.Create(dep.Revision); err != nil {
		LogVerbose("Failed to create package %s", pkg)
		return err
	}
	return nil
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
	deps     Dependencies
	gopath   string
	resolver RepoResolver
}

// NewDependencySaver builds a new dependencysaver to work in the
// specified gopath and resolve using the resolverfunc. A
// DependencySaver should generally only be used once. A
// DependencySaver will not attempt to load remote dependencies even
// if the resolverfunc can handle them. Deps that resolve using ignore
// will not be saved.
func NewDependencySaver(resolver RepoResolver, gopath string) *DependencySaver {
	return &DependencySaver{
		deps:     NewDependencies(),
		resolver: resolver,
		gopath:   gopath,
	}
}

// SavePackageRevision will attempt to load the revision of the
// package dep and add it to our Dependencies(). If errs contains any
// errors it will halt. If the package can not be loaded from gopath
// (is not a directory, does not exist, etc.) an error will be
// returned.
func (ds *DependencySaver) SavePackageRevision(pkg string) error {
	// Check if we can find this package
	s, err := os.Stat(PackageSource(ds.gopath, pkg))
	switch {
	case s != nil && !s.IsDir():
		return fmt.Errorf("Package %s is a file not a directory", pkg)
	case err != nil && os.IsNotExist(err):
		return fmt.Errorf("Package %s could not be found on disk", pkg)
	case err != nil:
		return err
	}

	// If we have already traversed a dep with a root for this
	// package just add the import path to it.
	if dep := ds.deps.DepForImportPath(pkg); dep != nil {
		dep.AddImportPaths(pkg)
		return nil
	}

	// Otherwise resolve it and save it
	vcs, err := ds.resolver.ResolveRepo(pkg, nil)
	if err != nil {
		return err
	}
	dep := &Dependency{}
	dep.AddImportPaths(pkg)
	dep.Root = vcs.GetRoot()
	dep.Revision, err = vcs.GetRev()
	if err != nil {
		return err
	}
	dep.SourcePath, err = vcs.GetSource()
	if err != nil {
		return err
	}
	return ds.deps.AddDependency(dep)
}

// Dependencies returns the resolved dependencies from dependency
// saver.
func (ds *DependencySaver) Dependencies() Dependencies {
	return ds.deps
}
