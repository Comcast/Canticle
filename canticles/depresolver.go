package canticles

import (
	"fmt"
	"os"
)

// A DependencySource represents the possible options to source a
// dependency from. Its possible revisions, remote sources, and other
// information like its on disk root, or errors resolving it.
type DependencySource struct {
	// Revisions specified by canticle files
	Revisions StringSet
	// OnDiskRevision for this VCS
	OnDiskRevision string
	// Sources specified for this VCS.
	Sources StringSet
	// OnDiskSource for this VCS.
	OnDiskSource string
	// Deps contained by this VCS system.
	Deps Dependencies
	// Root of the pacakges import path (prefix for all dep import paths).
	Root string
	// Err
	Err error
}

// NewDependencySource initalizes a DependencySource rooted at root on
// disk.
func NewDependencySource(root string) *DependencySource {
	return &DependencySource{
		Root:      root,
		Deps:      NewDependencies(),
		Revisions: NewStringSet(),
		Sources:   NewStringSet(),
	}
}

// AddCantSource adds a canticle source dep
func (d *DependencySource) AddCantSource(source *CanticleDependency, path string) {
	d.Revisions.Add(source.Revision)
	d.Sources.Add(source.SourcePath)
	dep := NewDependency(path)
	dep.Imports.Add(source.Root)
	d.Deps.AddDependency(dep)
}

// DependencySources represents a collection of dependencysources,
// including functionality to lookup deps that may be rooted in other
// deps.
type DependencySources struct {
	Sources []*DependencySource
}

// NewDependencySources with an iniital size for performance.
func NewDependencySources(size int) *DependencySources {
	return &DependencySources{make([]*DependencySource, 0, size)}
}

// DepSource returns the source for a dependency if its already
// present. That is if the deps importpath has a prefix in
// this collection.
func (ds *DependencySources) DepSource(importPath string) *DependencySource {
	for _, source := range ds.Sources {
		if source.Root == importPath || PathIsChild(source.Root, importPath) {
			return source
		}
	}
	return nil
}

// AddSource appends this DependencySource to our collection.
func (ds *DependencySources) AddSource(source *DependencySource) {
	ds.Sources = append(ds.Sources, source)
}

// String to pretty print this.
func (ds *DependencySources) String() string {
	str := ""
	for _, source := range ds.Sources {
		str += fmt.Sprintf("%s \n\tRevisions:%v OnDiskRevision:%s\n\tSources:%v OnDiskSource:%s\n\tDeps:", source.Root, source.Revisions, source.OnDiskRevision, source.Sources, source.OnDiskSource)
		for _, dep := range source.Deps {
			str += fmt.Sprintf("\n\t\t%+v", dep)
		}
		str += fmt.Sprintf("\n")
	}
	return str
}

// A SourcesResolver takes a set of dependencies and returns the
// possible sources and revisions for it (DependencySources) for it.
// Sources and Branches control whether the source (e.g. github.com)
// will be stored and whether brnaches of precise revisions will be
// saved.
type SourcesResolver struct {
	RootPath, Gopath  string
	Resolver          RepoResolver
	Branches, Sources bool
	CDepReader        CantDepReader
}

// ResolveSources for everything in deps, no dependency trees will be
// walked.
func (sr *SourcesResolver) ResolveSources(deps Dependencies) (*DependencySources, error) {
	sources := NewDependencySources(len(deps))
	for _, dep := range deps {
		LogVerbose("\tFinding source for %s", dep.ImportPath)
		// If we already have a source
		// for this dep just continue
		if source := sources.DepSource(dep.ImportPath); source != nil {
			LogVerbose("\t\tDep already added %s", dep.ImportPath)
			source.Deps.AddDependency(dep)
			continue
		}

		// Otherwise find the vcs root for it
		vcs, err := sr.Resolver.ResolveRepo(dep.ImportPath, nil)
		if err != nil {
			LogWarn("\t\tSkipping dep %+v, %s", dep, err.Error())
			continue
		}

		root := vcs.GetRoot()
		rootSrc := PackageSource(sr.Gopath, root)
		if rootSrc == sr.RootPath || PathIsChild(rootSrc, sr.RootPath) {
			LogVerbose("\t\tSkipping pkg %s since its vcs is at our save level", sr.RootPath)
			continue
		}
		source := NewDependencySource(root)

		var rev string
		if sr.Branches {
			rev, err = vcs.GetBranch()
			if err != nil {
				LogWarn("\t\tNo branch from vcs at %s %s", root, err.Error())
			}
		}
		if !sr.Branches || err != nil {
			rev, err = vcs.GetRev()
			if err != nil {
				return nil, fmt.Errorf("cant get revision from vcs at %s %s", root, err.Error())
			}
		}
		source.Revisions.Add(rev)
		source.OnDiskRevision = rev

		if sr.Sources {
			LogVerbose("\t\tGetting source for VCS: %s", root)
			vcsSource, err := vcs.GetSource()
			if err != nil {
				return nil, fmt.Errorf("cant get vcs source from vcs at %s %s", root, err.Error())
			}
			source.Sources.Add(vcsSource)
			source.OnDiskSource = vcsSource
		}
		source.Deps.AddDependency(dep)

		sources.AddSource(source)
	}

	// Resolve sources from importpaths, that is any canticle
	// files stored in a directory imported by our vcs
	for _, dep := range deps {
		if err := sr.resolveCantDeps(sources, dep.ImportPath); err != nil {
			return sources, err
		}
	}

	// Resolve any sources from our vcs roots, that is any
	// canticle files stored at the vcs route of a project.
	for _, source := range sources.Sources {
		if err := sr.resolveCantDeps(sources, source.Root); err != nil {
			return sources, err
		}
	}

	return sources, nil
}

func (sr *SourcesResolver) resolveCantDeps(sources *DependencySources, path string) error {
	cdeps, err := sr.CDepReader.CanticleDependencies(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, cdep := range cdeps {
		source := sources.DepSource(cdep.Root)
		if source == nil {
			continue
		}
		if !sr.Sources {
			cdep.SourcePath = ""
		}
		LogVerbose("\t\tAdding canticle source %+v", cdep)
		source.AddCantSource(cdep, path)
	}
	return nil
}
