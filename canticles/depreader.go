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
func (dr *DepReader) ReadCanticleDependencies(pkg string) ([]*CanticleDependency, error) {
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

// Read both the go and cant deps of a path.
func (dr *DepReader) ReadAllDeps(path string) (Dependencies, error) {
	allDeps := NewDependencies()
	// We only want to process directories, and ignore files
	pname, err := PackageName(dr.Gopath, path)
	if err != nil {
		return allDeps, err
	}
	// Attemp to read its canticle deps
	cdeps, err := dr.ReadCanticleDependencies(pname)
	if err != nil && !os.IsNotExist(err) {
		return allDeps, err
	}
	for _, cdep := range cdeps {
		if cdep.All {
			allDeps.AddDeps(cdep.Root)
		}
	}
	// If this is a dir attempt to read its deps, ignore all errors
	goDeps, err := dr.ReadGoRemoteDependencies(pname)
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
func (dr *DepReader) ReadGoRemoteDependencies(importPath string) ([]string, error) {
	pkg, err := LoadPackage(importPath, dr.Gopath)
	if err != nil {
		return []string{}, err
	}
	return pkg.RemoteImports(true), nil
}
