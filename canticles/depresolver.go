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
		Root:      root,
		Deps:      NewDependencies(),
		Revisions: NewStringSet(),
		Sources:   NewStringSet(),
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
		if source.Root == dep.ImportPath || PathIsChild(source.Root, dep.ImportPath) {
			return source
		}
	}
	return nil
}

func (ds *DependencySources) AddSource(source *DependencySource) {
	ds.Sources = append(ds.Sources, source)
}

func (ds *DependencySources) String() string {
	str := ""
	for _, source := range ds.Sources {
		str += fmt.Sprintf("%s: [\n\t%+v]\n", source.Root, source)
	}
	return str
}

type SourcesResolver struct {
	RootPath, Gopath string
	Resolver         RepoResolver
}

func (sr *SourcesResolver) ResolveSources(deps Dependencies) (*DependencySources, error) {
	sources := NewDependencySources(len(deps))
	LogWarn("Resolving version control systems for deps")
	for _, dep := range deps {
		LogVerbose("\tFinding source for %s", dep.ImportPath)
		// If we already have a source
		// for this dep just continue
		if source := sources.DepSource(dep); source != nil {
			LogVerbose("\t\tDep already added %s", dep.ImportPath)
			source.Deps.AddDependency(dep)
			continue
		}

		// Otherwise find the vcs root for it
		vcs, err := sr.Resolver.ResolveRepo(dep.ImportPath, nil)
		if err != nil {
			LogWarn("Skipping dep %+v, %s", dep, err.Error())
			continue
		}
		root := vcs.GetRoot()
		rootSrc := PackageSource(sr.Gopath, root)
		if rootSrc == sr.RootPath || PathIsChild(rootSrc, sr.RootPath) {
			LogWarn("Skipping pkg %s since its at our save level", sr.RootPath)
			continue
		}
		source := NewDependencySource(root)

		rev, err := vcs.GetRev()
		if err != nil {
			return nil, fmt.Errorf("cant get revision from vcs at %s %s", root, err.Error())
		}
		source.Revisions.Add(rev)
		source.OnDiskRevision = rev

		vcsSource, err := vcs.GetSource()
		if err != nil {
			return nil, fmt.Errorf("cant get vcs source from vcs at %s %s", root, err.Error())
		}
		source.Sources.Add(vcsSource)
		source.OnDiskSource = vcsSource
		source.Deps.AddDependency(dep)
		sources.AddSource(source)
	}
	return sources, nil
}
