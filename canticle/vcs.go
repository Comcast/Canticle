package canticle

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

	"golang.org/x/tools/go/vcs"
)

// A VCS has the ability to create and change the revision of a
// package. A VCS is generall resolved using a RepoDiscovery.
type VCS interface {
	Create(rev string) error
	SetRev(rev string) error
}

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

// PatchGitVCS will patch any git vcs.Cmd to use the correct tag lookup
// command.
func PatchGitVCS(v *vcs.Cmd) {
	if v.Cmd != "git" {
		return
	}
	v.TagLookupCmd = []vcs.TagCmd{
		{"rev-parse --quiet --verify {tag}", `(\w+)$`},
	}
}

// A LocalVCS uses packages and version control systems available at a
// local srcpath to control a local destpath (it copies the files over).
type LocalVCS struct {
	SrcPath  string
	DestPath string
	Cmd      *vcs.Cmd
}

// Create will copy (using a dir copier) the package from srcpath to
// destpath and then call set.
func (lv *LocalVCS) Create(rev string) error {
	dc := NewDirCopier(lv.SrcPath, lv.DestPath)
	dc.CopyDot = true
	if err := dc.Copy(); err != nil {
		return err
	}
	return lv.SetRev(rev)
}

// SetRev will use the LocalVCS's Cmd.TagSync method to change the
// revision of a repo if rev is not the empty string and Cmd is not
// nil.
func (lv *LocalVCS) SetRev(rev string) error {
	if lv.Cmd == nil || rev == "" {
		return nil
	}
	PatchGitVCS(lv.Cmd)
	return lv.Cmd.TagSync(lv.DestPath, rev)
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

func restoreWD(cwd string) {
	if err := os.Chdir(cwd); err != nil {
		log.Fatalf("Error restoring working directory: %s", err.Error())
	}

}

type localVCSCheckCmd struct {
	Vcs  string
	Cmd  string
	Args []string
}

var localVCSList = []localVCSCheckCmd{
	{"git", "git", []string{"rev-parse", "--is-inside-work-tree"}},
	{"svn", "svn", []string{"info"}},
	{"hg", "hg", []string{"root"}},
	{"bzr", "bzr", []string{"info"}},
}

// LocalVCSCheck will attempt to resolve a vcs.Cmd for a path p. If no
// resolution could occur nil, nil will be returned.
func LocalVCSCheck(p string) (*vcs.Cmd, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	if err := os.Chdir(p); err != nil {
		return nil, err
	}
	defer restoreWD(cwd)

	for _, c := range localVCSList {
		if _, err = exec.Command(c.Cmd, c.Args...).Output(); err == nil {
			return vcs.ByCmd(c.Vcs), nil
		}
	}
	return nil, nil
}

// PackageVCS wraps the underlying golang.org/x/tools/go/vcs to
// present the interface we need. It also implements the functionality
// necessary for SetRev to happen correctly.
type PackageVCS struct {
	Repo   *vcs.RepoRoot
	Gopath string
	cwd    string
}

func (pv *PackageVCS) cdRoot() {
	var err error
	pv.cwd, err = os.Getwd()
	if err != nil {
		log.Fatalf("Error getting working directory: %s", err.Error())
	}
	err = os.Chdir(path.Join(pv.Gopath, "src"))
	if err != nil {
		log.Fatalf("Error changing working directory: %s", err.Error())
	}
}

// Create clones the VCS into the location provided by Repo.Root
func (pv *PackageVCS) Create(rev string) error {
	pv.cdRoot()
	defer restoreWD(pv.cwd)
	v := pv.Repo.VCS
	if rev != "" {
		return v.Create(pv.Repo.Root, pv.Repo.Repo)
	}
	return v.CreateAtRev(pv.Repo.Root, pv.Repo.Repo, rev)
}

// SetRev changes the revision of the Repo.Root to the value
// provided. This also modifies the git based vcs to be able to deal
// with non named revisions (sigh).
func (pv *PackageVCS) SetRev(rev string) error {
	pv.cdRoot()
	defer restoreWD(pv.cwd)
	v := pv.Repo.VCS
	PatchGitVCS(v)
	return v.TagSync(pv.Repo.Root, rev)
}

// ErrorResolutionFailure will be returned if a RepoResolver could not
// resolve a VCS.
var ErrorResolutionFailure = errors.New("Discovery failed")

// RepoResolver provides the mechanisms for resolving a VCS from an
// importpath and sourcepath.
type RepoResolver interface {
	ResolveRepo(importPath, url string) (VCS, error)
}

// DefaultRepoResolver attempts to resolve a repo using the go
// vcs.RepoRootForImportPath semantics and guessing logic.
type DefaultRepoResolver struct {
	Gopath string
}

func (dr *DefaultRepoResolver) ResolveRepo(importPath, url string) (VCS, error) {
	// We guess our vcs based off our url path if present
	resolvePath := importPath
	if url != "" {
		resolvePath = url
	}

	fmt.Println("Guessing vcs for url: ", resolvePath)
	vcs.Verbose = Verbose

	// Next resort to the VCS (go get) guessing logic
	repo, err := vcs.RepoRootForImportPath(resolvePath, true)
	if err != nil {
		return nil, err
	}

	// If we found something return non nil
	repo.Root = importPath
	v := &PackageVCS{Repo: repo, Gopath: dr.Gopath}
	return v, nil
}

// RemoteRepoResolver attempts to resolve a repo using the internal
// guessing logic for Canticle.
type RemoteRepoResolver struct {
	Gopath string
}

func (rr *RemoteRepoResolver) ResolveRepo(importPath, url string) (VCS, error) {
	resolvePath := importPath
	if url != "" {
		resolvePath = url
	}
	// Attempt our internal guessing logic first
	guess, path, scheme := GuessVCS(resolvePath)
	if guess == nil {
		return nil, ErrorResolutionFailure
	}
	LogVerbose("Pinging path %s for vcs %v with scheme %s\n", path, guess, scheme)
	err := guess.Ping(scheme, path)
	if err != nil {
		return nil, ErrorResolutionFailure
	}
	guess.Scheme = []string{scheme}
	v := &PackageVCS{
		Repo: &vcs.RepoRoot{
			VCS:  guess,
			Repo: path,
			Root: importPath,
		},
		Gopath: rr.Gopath,
	}
	return v, nil
}

// LocalRepoResolver will attempt to find local copies of a repo in
// LocalPath (treating it like a gopath) and provide VCS systems for
// updating them in RemotePath (also treaded like a gopath).
type LocalRepoResolver struct {
	LocalPath  string
	RemotePath string
}

func (lr *LocalRepoResolver) ResolveRepo(importPath, url string) (VCS, error) {
	localPkg := path.Join(lr.LocalPath, "src", importPath)
	fmt.Printf("Localpkg: %s\n", localPkg)
	s, err := os.Stat(localPkg)
	switch {
	case err != nil:
		return nil, err
	case s != nil && s.IsDir():
		var cmd *vcs.Cmd
		cmd, err = LocalVCSCheck(localPkg)
		if err != nil {
			return nil, err
		}
		v := &LocalVCS{
			SrcPath:  localPkg,
			DestPath: path.Join(lr.RemotePath, "src", importPath),
			Cmd:      cmd,
		}
		return v, nil
	default:
		return nil, ErrorResolutionFailure
	}
}

// CompositeRepoResolver calls the repos in resolvers in order,
// discarding errors and returning the first VCS found.
type CompositeRepoResolver struct {
	Resolvers []RepoResolver
}

func (cr *CompositeRepoResolver) ResolveRepo(importPath, url string) (VCS, error) {
	for _, r := range cr.Resolvers {
		vcs, err := r.ResolveRepo(importPath, url)
		if vcs != nil && err == nil {
			return vcs, nil
		}
	}
	return nil, ErrorResolutionFailure
}

type resolve struct {
	v   VCS
	err error
}

// MemoizedRepoResolver remembers the results of previously attempted
// resolutions and will not attempt the twice.
type MemoizedRepoResolver struct {
	resolvedPaths map[string]*resolve
	resolver      RepoResolver
}

// NewMemoizedRepoResolver creates a memozied version of the passed in
// resolver.
func NewMemoizedRepoResolver(resolver RepoResolver) *MemoizedRepoResolver {
	return &MemoizedRepoResolver{
		resolvedPaths: make(map[string]*resolve),
		resolver:      resolver,
	}
}

func (mr *MemoizedRepoResolver) ResolveRepo(importPath, url string) (VCS, error) {
	r := mr.resolvedPaths[importPath]
	if r != nil {
		return r.v, r.err
	}

	v, err := mr.resolver.ResolveRepo(importPath, url)
	mr.resolvedPaths[importPath] = &resolve{v, err}
	return v, err
}
