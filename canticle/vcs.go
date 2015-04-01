package canticle

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"golang.org/x/tools/go/vcs"
)

// GitAtVCS creates a VCS cmd that supports the "git@blah.com:" syntax
func GitAtVCS() *vcs.Cmd {
	v := &vcs.Cmd{}
	*v = *vcs.ByCmd("git")
	v.CreateCmd = "clone git@{repo} {dir}"
	v.PingCmd = "ls-remote {scheme}@{repo}"
	v.Scheme = []string{"git"}
	v.PingCmd = "ls-remote {scheme}@{repo}"
	return v
}

// GuessVCS attemptxs to guess the VCS given a url. This mostly relies
// on the protocols like "ssh://" etc.
func GuessVCS(url string) (v *vcs.Cmd, repo, scheme string) {
	switch {
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

// PackageVCS wraps the underlying golang.org/x/tools/go/vcs to
// present the interface we need. It also implements the functionality
// necessary for SetRev to happen correctly.
type PackageVCS struct {
	Repo   *vcs.RepoRoot
	Goroot string
	cwd    string
}

func (pv *PackageVCS) cdRoot() {
	var err error
	pv.cwd, err = os.Getwd()
	if err != nil {
		log.Fatalf("Error getting working directory: %s", err.Error())
	}
	err = os.Chdir(path.Join(pv.Goroot, "src"))
	if err != nil {
		log.Fatalf("Error changing working directory: %s", err.Error())
	}
}

func (pv *PackageVCS) restoreWD() {
	if err := os.Chdir(pv.cwd); err != nil {
		log.Fatalf("Error restoring working directory: %s", err.Error())
	}

}

// Create clones the VCS into the location provided by Repo.Root
func (pv *PackageVCS) Create(rev string) error {
	pv.cdRoot()
	defer pv.restoreWD()
	if rev != "" {
		return pv.Repo.VCS.Create(pv.Repo.Root, pv.Repo.Repo)
	}
	return pv.Repo.VCS.CreateAtRev(pv.Repo.Root, pv.Repo.Repo, rev)
}

// SetRev changes the revision of the Repo.Root to the value
// provided. This also modifies the git based vcs to be able to deal
// with non named revisions (sigh).
func (pv *PackageVCS) SetRev(rev string) error {
	pv.cdRoot()
	defer pv.restoreWD()
	if pv.Repo.VCS.Cmd == "git" {
		pv.Repo.VCS.TagLookupCmd = []vcs.TagCmd{
			{"rev-parse --quiet --verify {tag}", `(\w+)$`},
		}
	}

	return pv.Repo.VCS.TagSync(pv.Repo.Root, rev)

}

// A VCS has the ability to create and change the revision of a
// package. A VCS is generall resolved using a RepoDiscovery.
type VCS interface {
	Create(rev string) error
	SetRev(rev string) error
}

// RepoDiscovery provides the mechanisms for resolving a VCS from an
// importpath and sourcepath. A VCS will only be resolved once for a
// given importpath. A cached copy will be returned there after. VCS
// items returned will be seeded with the correct goroot for use.
type RepoDiscovery struct {
	goroot      string
	resolvedVCS map[string]VCS
}

// NewRepoDiscovery creates a repodiscovery which will seed the VCS it
// discovers with goroot.
func NewRepoDiscovery(goroot string) *RepoDiscovery {
	return &RepoDiscovery{
		goroot:      goroot,
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
				Repo: &vcs.RepoRoot{
					VCS:  guess,
					Repo: path,
					Root: importPath,
				},
				Goroot: r.goroot,
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
	v := &PackageVCS{Repo: repo, Goroot: r.goroot}
	r.resolvedVCS[importPath] = v
	return v, nil
}
