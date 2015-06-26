package canticles

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// DepReader works in a particular gopath to read the
// dependencies of both Canticle and non-Canticle go packages.
type DepReader struct {
	Gopath string
}

// ReadAllCantDeps begins at a root folder of a vcs repo and traverses
// all folder and canticle deps listed. It will "swallow" any
// os.IsNotExist err's as well, possibly returning an empty set of
// deps.
func (dr *DepReader) ReadAllCantDeps(vcsRoot string) (Dependencies, error) {
	allDeps := NewDependencies()
	err := filepath.Walk(PackageSource(dr.Gopath, vcsRoot), func(p string, f os.FileInfo, err error) error {
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
		pname, err := PackageName(dr.Gopath, p)
		if err != nil {
			return err
		}
		// If this is a dir attempt to read its canticle deps
		deps, err := dr.ReadCanticleDependencies(pname)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		return allDeps.AddDependencies(deps)

	})
	return allDeps, err
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
