package canticles

import "fmt"

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

func NewDependencySource(root string) *DependencySource {
	return &DependencySource{
		Root: root,
		Deps: NewDependencies(),
	}
}

type DependencySources struct {
	Sources []*DependencySource
}

func NewDependencySources(size int) *DependencySources {
	return &DependencySources{make([]*DependencySource, 0, size)}
}

func (ds *DependencySources) DepSource(dep *Dependency) *DependencySource {
	for _, source := range ds.Sources {
		if PathIsChild(source.Root, dep.ImportPath) {
			return source
		}
	}
	return nil
}

func (ds *DependencySources) AddSource(source *DependencySource) {
	ds.Sources = append(ds.Sources, source)
}

type DependencyResolver struct {
	RootPath string
	Resolver RepoResolver
}

func (dr *DependencyResolver) ResolveDeps(deps Dependencies) (*DependencySources, error) {
	sources := NewDependencySources(len(deps))
	for _, dep := range deps {
		// If we already have a source
		// for this dep just continue
		if ds := sources.DepSource(dep); dep != nil {
			ds.Deps.AddDependency(dep)
			continue
		}

		// Otherwise find the vcs root for it
		vcs, err := dr.Resolver.ResolveRepo(dep.ImportPath, nil)
		if err != nil {
			return nil, err
		}
		root := vcs.GetRoot()
		if root == dr.RootPath {
			continue
		}
		source := NewDependencySource(root)

		rev, err := vcs.GetRev()
		if err != nil {
			return nil, fmt.Errorf("cant get revision from vcs at %s %s", root, err.Error())
		}
		source.Revisions.Add(rev)

		vcsSource, err := vcs.GetSource()
		if err != nil {
			return nil, fmt.Errorf("cant get vcs source from vcs at %s %s", root, err.Error())
		}
		source.Sources.Add(vcsSource)

		sources.AddSource(source)
	}
	return sources, nil
}
