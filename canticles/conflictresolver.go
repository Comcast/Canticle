package canticles

import "fmt"

type ConflictResolver interface {
	ResolveConflicts(deps *DependencySources) ([]*CanticleDependency, error)
}

type PreferLocalResolution struct {
}

func (pl *PreferLocalResolution) ResolveConflicts(deps *DependencySources) ([]*CanticleDependency, error) {
	cdeps := make([]*CanticleDependency, 0, len(deps.Sources))
	for _, source := range deps.Sources {
		if source.Err != nil {
			return cdeps, fmt.Errorf("cant resolve error saving dep %s", source.Err.Error())
		}
		cd := &CanticleDependency{
			Root:       source.Root,
			SourcePath: source.OnDiskSource,
			Revision:   source.OnDiskRevision,
		}
		cdeps = append(cdeps, cd)
	}
	return cdeps, nil
}

type PromptResolution struct {
	Printf func(format string, a ...interface{}) (n int, err error)
	Scanf  func(format string, a ...interface{}) (n int, err error)
}

func (pr PromptResolution) ResolveConflicts(deps *DependencySources) ([]*CanticleDependency, error) {
	cdeps := make([]*CanticleDependency, 0, len(deps.Sources))
	for _, dep := range deps.Sources {
		cd, err := pr.ResolveConflict(dep)
		if err != nil {
			return nil, err
		}
		cdeps = append(cdeps, cd)
	}
	return cdeps, nil
}

func (pr PromptResolution) ResolveConflict(dep *DependencySource) (*CanticleDependency, error) {
	cd := &CanticleDependency{Root: dep.Root}
	var err error
	size := dep.Revisions.Size()
	switch {
	case size > 1:
		cd.Revision, err = pr.SelectRevision(dep)
		if err != nil {
			return cd, err
		}
	case size == 1:
		cd.Revision = dep.Revisions.Array()[0]
	}
	size = dep.Sources.Size()
	switch {
	case size > 1:
		cd.SourcePath, err = pr.SelectSource(dep)
	case size == 1:
		cd.SourcePath = dep.Sources.Array()[0]
	}
	return cd, err
}

func (pr PromptResolution) SelectRevision(dep *DependencySource) (string, error) {
	return pr.ResolvePrompt(dep.Root, "revisions", dep.OnDiskRevision, dep.Revisions.Array())
}

func (pr PromptResolution) SelectSource(dep *DependencySource) (string, error) {
	return pr.ResolvePrompt(dep.Root, "sources", dep.OnDiskSource, dep.Sources.Array())
}

// ResolvePrompt is used to prompt a user for a resolution between
// multiple alternates, wth ondisk marking the currently select
// option.
// TODO: Add some sort of auto completion here
func (pr PromptResolution) ResolvePrompt(pkg, conflict, ondisk string, alts []string) (string, error) {
	pr.Printf("\nPackage %s has conflicting %s:\n", pkg, conflict)
	for _, rev := range alts {
		if rev == ondisk {
			pr.Printf("%s %s\n", rev, "(current)")
			continue
		}
		pr.Printf("%s\n", rev)
	}
	pr.Printf("Selection %s: ", conflict)
	var choice string
	_, err := pr.Scanf("%s", &choice)
	return choice, err
}
