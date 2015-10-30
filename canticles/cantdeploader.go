package canticles

import (
	"fmt"
	"sync"
)

// A CantDepReader should return the CanticleDependencies of a
// package. Usually stored in the Canticle file.
type CantDepReader interface {
	CanticleDependencies(pkg string) ([]*CanticleDependency, error)
}

// A CanticleDepLoader is used to fetch the dependencies of a for a
// set of CanticleDependencies. It uses a reader for fetchpath to read
// the dependencies in a path and a resolver to resolve the vcs for
// each path and actually fetch the dep. Update can be set to udpate
// branches for a dep.
type CanticleDepLoader struct {
	Reader   CantDepReader
	Resolver RepoResolver
	Gopath   string
	Update   bool
	updated  map[string]string
}

// FetchPath fetches the dependencies in a Canticle file at path. It
// will return an array of errors encountered while fetching those
// deps.
func (cdl *CanticleDepLoader) FetchPath(path string) []error {
	LogVerbose("Reading %s canticle deps", path)
	pkg, err := PackageName(cdl.Gopath, path)
	if err != nil {
		return []error{err}
	}
	cdeps, err := cdl.Reader.CanticleDependencies(pkg)
	if err != nil {
		return []error{fmt.Errorf("cant fetch package %s couldn't read cant file %s", pkg, err.Error())}
	}
	LogVerbose("Read package canticle %s deps", pkg)
	return cdl.FetchDeps(cdeps...)
}

type update struct {
	cdep *CanticleDependency
	rev  string
	err  error
}

// FetchDeps will fetch all of the cdeps passed to it in parallel and
// return an array of encountered errors.
func (cdl *CanticleDepLoader) FetchDeps(cdeps ...*CanticleDependency) []error {
	cdl.updated = make(map[string]string)
	results := make(chan update)
	var wg sync.WaitGroup
	var errors []error
	for _, cdep := range cdeps {
		wg.Add(1)
		go func(cdep *CanticleDependency) {
			rev, err := FetchDep(cdl.Resolver, cdep, cdl.Update)
			results <- update{cdep, rev, err}
			wg.Done()
		}(cdep)
	}
	go func() {
		wg.Wait()
		close(results)
	}()
	for result := range results {
		if result.err != nil {
			errors = append(errors, result.err)
		}
		if result.rev != "" {
			cdl.updated[result.cdep.Root] = result.rev
		}
	}
	return errors
}

// Updated returns a map of repo roots that where updated by the last
// fetch deps/fetchpath call and the resulting info from the update.
func (cdl *CanticleDepLoader) Updated() map[string]string {
	return cdl.updated
}

// FetchDep fetchs a single canticle dep using the resolver. If update
// is true it will update the vcs branch to cdep.Revision. If not
// updated the rev string will be the empty string.
func FetchDep(resolver RepoResolver, cdep *CanticleDependency, update bool) (string, error) {
	LogInfo("Resolving repo for cdep %+v", cdep)
	vcs, err := resolver.ResolveRepo(cdep.Root, cdep)
	if err != nil {
		return "", fmt.Errorf("cant create vcs for %v because %s", cdep, err.Error())
	}
	LogInfo("Fetching cdep %+v", cdep)
	if err := vcs.Create(cdep.Revision); err != nil {
		return "", fmt.Errorf("cant fetch repo %s because %s", cdep.Root, err.Error())
	}
	if update {
		LogVerbose("Updating cdep %+v", cdep)
		updated, res, err := vcs.UpdateBranch(cdep.Revision)
		if !updated {
			res = ""
		}
		if err != nil {
			return res, fmt.Errorf("cant update repo %s because %s", cdep.Root, err.Error())
		}
		return res, nil
	}

	return "", nil
}
