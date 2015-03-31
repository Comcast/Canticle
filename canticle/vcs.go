package canticle

import (
	"fmt"
	"strings"

	"golang.org/x/tools/go/vcs"
)

// GitATVCS creates a VCS cmd that supports the "git@blah.com:" syntax
func GitAtVCS() *vcs.Cmd {
	v := &vcs.Cmd{}
	*v = *vcs.ByCmd("git")
	v.CreateCmd = "clone git@{repo} {dir}"
	v.PingCmd = "ls-remote {scheme}@{repo}"
	v.Scheme = []string{"git"}
	v.PingCmd = "ls-remote {scheme}@{repo}"
	return v
}

// Attempt to guess the VCS given a url. This mostly relies on the
// protocols like "ssh://" etc.
func GuessVCS(url string) (v *vcs.Cmd, repo, scheme string) {
	switch {
	case strings.HasPrefix(url, "https://github.com"):
		return vcs.ByCmd("git"), strings.TrimPrefix(url, "https://"), "https"
	case strings.HasPrefix(url, "git+ssh://"):
		return vcs.ByCmd("git"), strings.TrimPrefix(url, "git+ssh://"), "git+ssh"
	case strings.HasPrefix(url, "git://"):
		return vcs.ByCmd("git"), strings.TrimPrefix(url, "git://"), "git+ssh"
	case strings.HasPrefix(url, "git@"):
		return GitAtVCS(), strings.TrimPrefix(url, "git@"), "git"
	case strings.HasPrefix(url, "ssh://hg@"):
		return vcs.ByCmd("hg"), strings.TrimPrefix(url, "ssh://"), "ssh"
	case strings.HasPrefix(url, "svn://"):
		return vcs.ByCmd("svn"), strings.TrimPrefix(url, "svn://"), "svn"
	case strings.HasPrefix(url, "bzr://"):
		return vcs.ByCmd("bzr"), strings.TrimPrefix(url, "bzr://"), "bzr"
	default:
		return nil, "", ""
	}
}

type PackageVCS struct {
	repo *vcs.RepoRoot
}

func (pv *PackageVCS) Create(rev string) error {
	if rev != "" {
		return pv.repo.VCS.Create(pv.repo.Root, pv.repo.Repo)
	}
	return pv.repo.VCS.CreateAtRev(pv.repo.Root, pv.repo.Repo, rev)
}

func (pv *PackageVCS) SetRev(rev string) error {
	return nil
}

type VCS interface {
	Create(rev string) error
	SetRev(rev string) error
}

type RepoDiscovery struct {
	resolvedVCS map[string]VCS
}

func NewRepoDiscovery() *RepoDiscovery {
	return &RepoDiscovery{
		resolvedVCS: make(map[string]VCS),
	}
}

// ResolveRepo uses a URL and an import path to guess
// a correct location and vcs for given import path. If the URL does
// not return a VCS from GuessVCS then it falls back on
// vcs.RepoRootForImportPath.
func (r *RepoDiscovery) ResolveRepo(importPath, url string) (VCS, error) {
	if vcs, found := r.resolvedVCS[importPath]; found {
		return vcs, nil
	}

	// We guess our vcs based off our url path if present
	resolvePath := importPath
	if url != "" {
		resolvePath = url
	}

	fmt.Println("Guessing vcs for url: ", resolvePath)
	vcs.Verbose = true

	// Attempt our internal guessing logic first
	guess, path, scheme := GuessVCS(resolvePath)
	if guess != nil {
		fmt.Printf("Pinging path %s for vcs %v with scheme %s\n", path, guess, scheme)
		err := guess.Ping(scheme, path)
		if err == nil {
			guess.Scheme = []string{scheme}
			v := &PackageVCS{
				&vcs.RepoRoot{
					VCS:  guess,
					Repo: path,
					Root: importPath,
				},
			}
			r.resolvedVCS[importPath] = v
			return v, nil
		}
		fmt.Println("Ping vcs err: ", err.Error())
	}

	// Next resort to the VCS (go get) guessing logic
	repo, err := vcs.RepoRootForImportPath(resolvePath, true)
	if err != nil {
		return nil, err
	}

	// If we found something return non nil
	repo.Root = importPath
	v := &PackageVCS{repo}
	r.resolvedVCS[importPath] = v
	return v, nil
}
