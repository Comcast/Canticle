package canticles

import "fmt"

// A Dependency defines all information about this package.
type Dependency struct {
	// ImportPath for this string as it would appear in a go file.
	ImportPath string
	// ImportedFrom is a list of packages which import
	// this dependency.
	ImportedFrom StringSet
	// Imports is the set of remote imports for this dep.
	Imports StringSet
	// Attempt to read the package caused an error.
	Err error
}

func NewDependency(importPath string) *Dependency {
	return &Dependency{
		ImportedFrom: NewStringSet(),
		Imports:      NewStringSet(),
		ImportPath:   importPath,
	}
}

// Dependencies is the set of Dependencies for a package. This is
// stored as a map from Root to Dependency.
type Dependencies map[string]*Dependency

// NewDependencies creates a new empty dependencies structure.
func NewDependencies() Dependencies {
	return make(map[string]*Dependency)
}

func (d Dependencies) Dependency(importPath string) *Dependency {
	return d[importPath]
}

// AddDependencies ranges of the dependencies in deps and calls
// AddDependency on them.
func (d Dependencies) AddDependencies(deps Dependencies) {
	for _, dep := range deps {
		d.AddDependency(dep)
	}
}

// AddDependency adds a
func (d Dependencies) AddDependency(dep *Dependency) {
	already := d[dep.ImportPath]
	if already == nil {
		d[dep.ImportPath] = dep
		return
	}

	already.Err = dep.Err
	already.ImportedFrom.Union(dep.ImportedFrom)
	already.Imports.Union(dep.Imports)
}

func (d Dependencies) AddDeps(deps ...string) {
	for _, dep := range deps {
		if d[dep] == nil {
			d[dep] = NewDependency(dep)
		}
	}
}

// String will print this out as newline seperated %+v values.
func (d Dependencies) String() string {
	str := ""
	for path, dep := range d {
		str += fmt.Sprintf("%s: %+v\n", path, dep)
	}
	return str
}

type CanticleDependency struct {
	// SourcePath is the source of the dependencies VCS
	SourcePath string `json:",omitempty"`
	// Root is the root VCS for the import paths. It must be
	// a prefix of all import paths.
	Root string
	// Revision is the VCS specific commit id
	Revision string `json:",omitempty"`
	// All means walks this VCS from the root for nonhidden files. This will save and
	// fetch the subdirs of package.
	All bool `json:",omitempty"`
}

type CanticleDependencies []*CanticleDependency

func (cd CanticleDependencies) Len() int {
	return len(cd)
}

func (cd CanticleDependencies) Less(i, j int) bool {
	return cd[i].Root < cd[j].Root
}

func (cd CanticleDependencies) Swap(i, j int) {
	cd[i], cd[j] = cd[j], cd[i]
}
