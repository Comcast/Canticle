package canticle

import (
	"encoding/json"
	"os"
	"path"
)

// DepReader works in a particular gopath to read the
// dependencies of both Canticle and non-Canticle go packages.
type DepReader struct {
	Gopath string
}

// ReadCanticleDependencies returns the dependencies listed in the
// packages Canticle file. Dependencies will never be nil.
func (dr *DepReader) ReadCanticleDependencies(pkg string) (Dependencies, error) {
	deps := NewDependencies()
	f, err := os.Open(path.Join(PackageSource(dr.Gopath, pkg), "Canticle"))
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

// ReadRemoteDependencies reads the dependencies for package p listed
// as imports in *.go files, including tests, and returns the result.
func (dr *DepReader) ReadRemoteDependencies(importPath string) ([]string, error) {
	pkg, err := LoadPackage(importPath, dr.Gopath)
	if err != nil {
		return []string{}, err
	}
	return pkg.RemoteImports(true), nil
}
