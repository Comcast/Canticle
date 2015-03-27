package canticle

import (
	"encoding/json"
	"io/ioutil"
	"path"
)

// A CanticleDependency defines deps for this package to pull
type CanticleDependency struct {
	ImportPath string
	SourcePath string `json:,ommitempty`
	VCS        string `json:,ommitempty`
	Revision   string `json:,ommitempty`
	Comment    string `json:,ommitempty`
}

type CanticleDependencies map[string]*CanticleDependency

func (c CanticleDependencies) SourcePath(p string) string {
	if c[p] != nil {
		return c[p].SourcePath
	}
	return ""
}

func (c CanticleDependencies) Revision(p string) string {
	if c[p] != nil {
		return c[p].Revision
	}
	return ""
}

func LoadDependencies(p, gopath string) (CanticleDependencies, error) {
	c, err := ioutil.ReadFile(path.Join(gopath, "src", p, "Canticle"))
	if err != nil {
		return nil, err
	}

	d := make([]*CanticleDependency, 0)
	if err := json.Unmarshal(c, &d); err != nil {
		return nil, err
	}

	deps := make(CanticleDependencies, len(d))
	for _, dep := range d {
		deps[dep.ImportPath] = dep
	}
	return deps, nil
}

func SaveDependencies() CanticleDependencies {
	return nil
}
