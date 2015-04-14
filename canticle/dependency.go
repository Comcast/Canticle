package canticle

import (
	"encoding/json"
	"fmt"
	"strings"
)

// A Dependency defines all information about this package.
type Dependency struct {
	// ImportPaths contains the import paths of a given repo. Each
	// ImportPath must be a child of Root.
	ImportPaths []string
	// SourcePath is the source of the dependencies VCS
	SourcePath string `json:",omitempty"`
	// Root is the root VCS for the import paths. It must be
	// a prefix of all import paths.
	Root string `json:",omitempty"`
	// Revision is the VCS specific commit id
	Revision string `json:",omitempty"`
	// A comment to be kept about this repo
	Comment string `json:",omitempty"`
}

// AddImportPaths p if not already present
func (d *Dependency) AddImportPaths(p ...string) {
	d.ImportPaths = MergeStringsAsSet(d.ImportPaths, p...)
}

// Dependencies is the set of Dependencies for a package. This is
// stored as a map from Root to Dependency.
type Dependencies map[string]*Dependency

// NewDependencies creates a new empty dependencies structure.
func NewDependencies() Dependencies {
	return make(map[string]*Dependency)
}

// DepForImportPath returns a dep if any dep root is a prefix for p.
func (d Dependencies) DepForImportPath(p string) *Dependency {
	for _, dep := range d {
		if dep.Root != "" && strings.HasPrefix(p, dep.Root) {
			return dep
		}
	}
	return nil
}

// AddDependencies ranges of the dependencies in deps and calls
// AddDependency on them.
func (d Dependencies) AddDependencies(deps Dependencies) error {
	for _, dep := range deps {
		if err := d.AddDependency(dep); err != nil {
			return err
		}
	}

	return nil
}

// AddDependency adds a dependency if its root is not already
// present. If the root is present import pths will be added to the
// dep. If a conflict is detected the following merge strategy will be
// taken:
//
// * A non blank ("") revision will always be prefered to a blank one
// * A non blank ("") source path will always be prefered to a blank one
// * If there is a conflicting revision the current one will be taken, an
//   error will be returned
// * If there is a conflicting sourcepath the current one will be taken, an error will be returned
// * Comments are never merged
func (d Dependencies) AddDependency(dep *Dependency) error {
	already := d[dep.Root]
	if already == nil {
		d[dep.Root] = dep
		return nil
	}

	already.AddImportPaths(dep.ImportPaths...)
	switch {
	case already.Revision == "" && dep.Revision != "":
		already.Revision = dep.Revision
	case already.SourcePath == "" && dep.SourcePath != "":
		already.SourcePath = dep.SourcePath
	case dep.Revision != "" && already.Revision != dep.Revision:
		return fmt.Errorf("Dep %+v has conflicting revisions %s and %s\n", already, already.Revision, dep.Revision)
	case dep.SourcePath != "" && already.SourcePath != dep.SourcePath:
		return fmt.Errorf("Dep %+v has conflicting source %s and %s\n", already, already.SourcePath, dep.SourcePath)
	}

	return nil
}

// RemoveRoot will delete a dep based on its root.
func (d Dependencies) RemoveRoot(root string) {
	delete(d, root)
}

// Dependency returns nil if importPath is not present.
func (d Dependencies) Dependency(root string) *Dependency {
	return d[root]
}

// String will print this out as newline seperated %+v values.
func (d Dependencies) String() string {
	str := ""
	for _, dep := range d {
		str += fmt.Sprintf("%+v\n", dep)
	}
	return str
}

// MarshalJSON will encode this as a JSON array.
func (d Dependencies) MarshalJSON() ([]byte, error) {
	deps := make([]*Dependency, 0, len(d))
	for _, dep := range d {
		deps = append(deps, dep)
	}
	return json.Marshal(deps)
}

// UnmarshalJSON will decode this from a JSON array.
func (d Dependencies) UnmarshalJSON(data []byte) error {
	var deps []*Dependency
	if err := json.Unmarshal(data, &deps); err != nil {
		return err
	}
	for _, dep := range deps {
		d.AddDependency(dep)
	}
	return nil
}
