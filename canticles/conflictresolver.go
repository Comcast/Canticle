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
}

func (pr *PromptResolution) ResolveConflicts(deps *DependencySources) ([]*CanticleDependency, error) {
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

func (pr *PromptResolution) ResolveConflict(dep *DependencySource) (*CanticleDependency, error) {
	cd := &CanticleDependency{Root: dep.Root}
	var err error
	if dep.Revisions.Size() > 1 {
		cd.Revision, err = pr.SelectRevision(dep)
		if err != nil {
			return cd, err
		}
	} else {
		cd.Revision = dep.Revisions.Array()[0]
	}
	if dep.Sources.Size() > 1 {
		cd.Revision, err = pr.SelectSource(dep)
	} else {
		cd.SourcePath = dep.Sources.Array()[0]
	}
	return cd, err
}

func (pr *PromptResolution) SelectRevision(dep *DependencySource) (string, error) {
	return ResolvePrompt(dep.Root, "revisions", dep.OnDiskRevision, dep.Revisions.Array())
}

func (pr *PromptResolution) SelectSource(dep *DependencySource) (string, error) {
	return ResolvePrompt(dep.Root, "sources", dep.OnDiskSource, dep.Sources.Array())
}

// ResolvePrompt is used to prompt a user for a resolution between
// multiple alternates, wth ondisk marking the currently select
// option.
// TODO: Add some sort of auto completion here
func ResolvePrompt(pkg, conflict, ondisk string, alts []string) (string, error) {
	fmt.Printf("\nPackage %s has conflicting %s:\n", pkg, conflict)
	for _, rev := range alts {
		if rev == ondisk {
			fmt.Println(rev, "(current)")
			continue
		}
		fmt.Println(rev)
	}
	fmt.Printf("Selection %s: ", conflict)
	var choice string
	_, err := fmt.Scanf("%s", &choice)
	return choice, err
}
