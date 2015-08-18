package canticles

import (
	"encoding/json"
	"os"
)

// DepReader works in a particular gopath to read the
// dependencies of both Canticle and non-Canticle go packages.
type DepReader struct {
	Gopath string
}

// ReadCanticleDependencies returns the dependencies listed in the
// packages Canticle file. Dependencies will never be nil.
func (dr *DepReader) CanticleDependencies(pkg string) ([]*CanticleDependency, error) {
	var deps []*CanticleDependency
	f, err := os.Open(DependencyFile(PackageSource(dr.Gopath, pkg)))
	if err != nil {
		return deps, err
	}
	LogVerbose("Reading canticle file: %s", f.Name())
	defer f.Close()
	d := json.NewDecoder(f)
	if err := d.Decode(&deps); err != nil {
		return deps, err
	}
	return deps, nil
}

/*
func (dr *DepReader) ReadRecursive(path string) (Dependencies, error) {
	allDeps := NewDependencies()
	err := filepath.Walk(PackageSource(dr.Gopath, root), func(p string, f os.FileInfo, err error) error {
		// Go src dirs don't have dot prefixes
		if strings.HasPrefix(filepath.Base(p), ".") {
			if f.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if err != nil {
			return err
		}
		// We only want to process directories, and ignore files
		if !f.IsDir() {
			return nil
		}
		deps, err := dr.ReadAllDeps(p)
		if err != nil {
			return err
		}
		allDeps.AddDependencies(deps)
		return nil

	})
	return allDeps, err

}
*/
func (dr *DepReader) AllImports(path string) ([]string, error) {
	deps, err := dr.AllDeps(path)
	if err != nil {
		return nil, err
	}

	imports := make([]string, 0, len(deps))
	for _, dep := range deps {
		imports = append(imports, dep.ImportPath)
	}
	return imports, nil
}

// Read both the go and cant deps of a path.
func (dr *DepReader) AllDeps(path string) (Dependencies, error) {
	allDeps := NewDependencies()
	// We only want to process directories, and ignore files
	pname, err := PackageName(dr.Gopath, path)
	if err != nil {
		return allDeps, err
	}
	// Attemp to read its canticle deps
	cdeps, err := dr.CanticleDependencies(pname)
	if err != nil && !os.IsNotExist(err) {
		return allDeps, err
	}
	for _, cdep := range cdeps {
		if cdep.All {
			allDeps.AddDeps(cdep.Root)
		}
	}
	// If this is a dir attempt to read its deps, ignore if it has
	// no go files
	goDeps, err := dr.GoRemoteDependencies(pname)
	if err != nil {
		switch e := err.(type) {
		case *PackageError:
			if !e.IsNoBuildable() {
				return allDeps, err
			}
		default:
			return allDeps, err
		}
	}
	allDeps.AddDeps(goDeps...)
	return allDeps, nil
}

// ReadGoRemoteDependencies reads the dependencies for package p listed
// as imports in *.go files, including tests, and returns the result.
func (dr *DepReader) GoRemoteDependencies(importPath string) ([]string, error) {
	pkg, err := LoadPackage(importPath, dr.Gopath)
	if err != nil {
		return []string{}, err
	}
	return pkg.RemoteImports(true), nil
}
