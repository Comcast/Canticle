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
	Update   bool
	updated  map[string]string
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
	cdl.updated = make(map[string]string)
	for _, cdep := range cdeps {
		LogVerbose("Resolving repo for cdep %+v", cdep)
		vcs, err := cdl.Resolver.ResolveRepo(cdep.Root, cdep)
		if err != nil {
			return fmt.Errorf("cant create vcs for %v because %s", cdep, err.Error())
		}
		LogVerbose("Fetching cdep %+v", cdep)
		if err := vcs.Create(cdep.Revision); err != nil {
			return fmt.Errorf("cant fetch repo %s because %s", cdep.Root, err.Error())
		}
		if cdl.Update {
			LogVerbose("Updating cdep %+v", cdep)
			updated, res, err := vcs.UpdateBranch(cdep.Revision)
			if err != nil {
				return fmt.Errorf("cant update repo %s because %s", cdep.Root, err.Error())
			}
			if updated {
				cdl.updated[cdep.Root] = res
			} else {
				LogVerbose("Did not update %s", res)
			}
		}
		LogVerbose("\n")
	}
	return nil
}

// Updated returns a map of repo roots that where updated by the last
// fetch deps/fetchpath call and the resulting info from the update.
func (cdl *CanticleDepLoader) Updated() map[string]string {
	return cdl.updated
}
