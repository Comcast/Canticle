package canticles

import (
	"fmt"
	"os"
)

type CantDepReader interface {
	CanticleDependencies(pkg string) ([]*CanticleDependency, error)
}

type CanticleDepLoader struct {
	Reader   CantDepReader
	Resolver RepoResolver
	Gopath   string
}

func (cdl *CanticleDepLoader) FetchPath(path string) error {
	LogVerbose("Reading %s canticle deps", path)
	pkg, err := PackageName(cdl.Gopath, path)
	if err != nil {
		return err
	}
	cdeps, err := cdl.Reader.CanticleDependencies(pkg)
	switch {
	case err != nil && !os.IsNotExist(err):
		return fmt.Errorf("cant fetch package %s couldn't read cant file %s", pkg, err.Error())
	case err != nil && os.IsNotExist(err):
		return fmt.Errorf("cant read %s deps %s", pkg, err.Error())
	}
	LogVerbose("Read package canticle %s deps", pkg)
	return cdl.FetchDeps(cdeps...)
}

func (cdl *CanticleDepLoader) FetchDeps(cdeps ...*CanticleDependency) error {
	for _, cdep := range cdeps {
		LogVerbose("Resolving repo for cdep %+v", cdep)
		vcs, err := cdl.Resolver.ResolveRepo(cdep.Root, cdep)
		if err != nil {
			return fmt.Errorf("%v version control %s", cdep, err.Error())
		}
		LogVerbose("Fetching cdep %+v", cdep)
		if err := vcs.Create(cdep.Revision); err != nil {
			return fmt.Errorf("failed to fetch because %s", err.Error())
		}
	}
	return nil
}
