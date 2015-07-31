package canticles

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"

	"golang.org/x/tools/go/vcs"
)

// A VCS has the ability to create and change the revision of a
// package. A VCS is generall resolved using a RepoDiscovery.
type VCS interface {
	Create(rev string) error
	SetRev(rev string) error
	GetRev() (string, error)
	GetSource() (string, error)
	GetRoot() string
}

// GitAtVCS creates a VCS cmd that supports the "git@blah.com:" syntax
func GitAtVCS() *vcs.Cmd {
	v := &vcs.Cmd{}
	*v = *vcs.ByCmd("git")
	v.Name = "Git@"
	v.CreateCmd = "clone {repo} {dir}"
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

// A VCSCmd is used to run a VCS command for a repo
type VCSCmd struct {
	Name       string
	Cmd        string
	Args       []string
	ParseRegex *regexp.Regexp
}

// Exec executes this command with its arguments and parses them using
// regexp. Return an error if the command generates an error or we can
// not parse the results.
func (vc *VCSCmd) Exec(repo string) (string, error) {
	LogVerbose("Running rev command: %s %v in dir %s", vc.Cmd, vc.Args, repo)
	cmd := exec.Command(vc.Cmd, vc.Args...)
	cmd.Dir = repo
	result, err := cmd.CombinedOutput()
	resultTrim := strings.TrimSpace(string(result))
	rev := vc.ParseRegex.FindSubmatch([]byte(resultTrim))
	switch {
	case err != nil:
		return "", fmt.Errorf("Error getting revision %s", result)
	case result == nil:
		return "", errors.New("Error vcs returned no info for revision")
	case rev == nil:
		return "", fmt.Errorf("Error parsing cmd result:\n%s", string(result))
	default:
		return string(rev[1]), nil
	}
}

var (
	// GitRevCmd attempts to pull the current git from a git
	// repo. It will fail if the work tree is "dirty".
	GitRevCmd = &VCSCmd{
		Name:       "Git",
		Cmd:        "git",
		Args:       []string{"rev-parse", "HEAD"},
		ParseRegex: regexp.MustCompile(`(\S+)`),
	}
	// SvnRevCmd attempts to pull the current svnversion from a svn
	// repo.
	SvnRevCmd = &VCSCmd{
		Name:       "Subversion",
		Cmd:        "svnversion",
		ParseRegex: regexp.MustCompile(`^(\S+)$`), // svnversion doesn't have a bad exitcode if not in svndir
	}
	// BzrRevCmd attempts to pull the current revno from a Bazaar
	// repo.
	BzrRevCmd = &VCSCmd{
		Name:       "Bazaar",
		Cmd:        "bzr",
		Args:       []string{"revno"},
		ParseRegex: regexp.MustCompile(`(\S+)`),
	}
	// HgRevCmd attempts to pull the current node from a Mercurial
	// repo.
	HgRevCmd = &VCSCmd{
		Name:       "Mercurial",
		Cmd:        "hg",
		Args:       []string{"log", "--template", "{node}"},
		ParseRegex: regexp.MustCompile(`(\S+)`),
	}
	// RevCmds is a map of cmd (git, svn, etc.) to
	// the cmd to parse its revision.
	RevCmds = map[string]*VCSCmd{
		GitRevCmd.Name: GitRevCmd,
		SvnRevCmd.Name: SvnRevCmd,
		BzrRevCmd.Name: BzrRevCmd,
		HgRevCmd.Name:  HgRevCmd,
	}

	// GitRemoteCmd attempts to pull the origin of a git repo.
	GitRemoteCmd = &VCSCmd{
		Name:       "Git",
		Cmd:        "git",
		Args:       []string{"ls-remote", "--get-url", "origin"},
		ParseRegex: regexp.MustCompile(`^(.+)$`),
	}
	// SvnRemoteCmd attempts to pull the origin of a svn repo.
	SvnRemoteCmd = &VCSCmd{
		Name:       "Subversion",
		Cmd:        "svn",
		Args:       []string{"info"},
		ParseRegex: regexp.MustCompile(`^URL: (.+)$`), // svnversion doesn't have a bad exitcode if not in svndir
	}
	// HgRemoteCmd attempts to pull the current default paths from
	// a Mercurial repo.
	HgRemoteCmd = &VCSCmd{
		Name:       "Mercurial",
		Cmd:        "hg",
		Args:       []string{"paths", "default"},
		ParseRegex: regexp.MustCompile(`(.+)`),
	}
	// RemoteCmds is a map of cmd (git, svn, etc.) to
	// the cmd to parse its revision.
	RemoteCmds = map[string]*VCSCmd{
		GitRemoteCmd.Name: GitRemoteCmd,
		SvnRemoteCmd.Name: SvnRemoteCmd,
		HgRemoteCmd.Name:  HgRemoteCmd,
	}
)

// A LocalVCS uses packages and version control systems available at a
// local srcpath to control a local destpath (it copies the files over).
type LocalVCS struct {
	Package       string
	Root          string
	SrcPath       string
	Cmd           *vcs.Cmd
	CurrentRevCmd *VCSCmd // CurrentRevCommand to check the current revision for sourcepath.
	RemoteCmd     *VCSCmd // RemoteCmd to obtain the upstream (remote) for a repo
}

// NewLocalVCS returns a a LocalVCS with CurrentRevCmd initialized
// from the cmd's name using RevCmds and RemoteCmd from RemoteCmds.
func NewLocalVCS(pkg, root, srcPath string, cmd *vcs.Cmd) *LocalVCS {
	return &LocalVCS{
		Package:       pkg,
		Root:          root,
		SrcPath:       srcPath,
		Cmd:           cmd,
		CurrentRevCmd: RevCmds[cmd.Name],
		RemoteCmd:     RemoteCmds[cmd.Name],
	}
}

// Create will copy (using a dir copier) the package from srcpath to
// destpath and then call set.
func (lv *LocalVCS) Create(rev string) error {
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
	return lv.Cmd.TagSync(lv.Root, rev)
}

// GetRev will return current revision of the local repo.  If the
// local package is not under a VCS it will return nil, nil.  If the
// vcs can not query the version it will return nil and an error.
func (lv *LocalVCS) GetRev() (string, error) {
	if lv.CurrentRevCmd == nil || lv.Cmd == nil {
		return "", nil
	}
	return lv.CurrentRevCmd.Exec(PackageSource(lv.SrcPath, lv.Root))

}

// GetSource on a LocalVCS will attempt to determine the local repos
// upstream source. See the RemoteCmd for each VCS for behavior.
func (lv *LocalVCS) GetSource() (string, error) {
	if lv.RemoteCmd == nil {
		return "", nil
	}
	return lv.RemoteCmd.Exec(PackageSource(lv.SrcPath, lv.Root))
}

// GetRoot on a LocalVCS will return PackageName for SrcPath
func (lv *LocalVCS) GetRoot() string {
	return lv.Root
}

// VCSType represents a prefix to look for, a scheme to ping a path
// with and a VCS command to do the pinging.
type VCSType struct {
	Prefix string
	Scheme string
	VCS    *vcs.Cmd
}

// VCSTypes is the list of VCSType used by GuessVCS
var VCSTypes = []VCSType{
	{"git+ssh://", "git+ssh", vcs.ByCmd("git")},
	{"git://", "git", vcs.ByCmd("git")},
	{"git@", "git", GitAtVCS()},
	{"ssh://hg@", "ssh", vcs.ByCmd("hg")},
	{"svn://", "svn", vcs.ByCmd("svn")},
	{"bzr://", "bzr", vcs.ByCmd("bzr")},
}

// GuessVCS attempts to guess the VCS given a url. This uses the
// VCSTypes array, checking for prefixes that match and attempting to
// ping the VCS with the given scheme
func GuessVCS(url string) *vcs.Cmd {
	for _, vt := range VCSTypes {
		if !strings.HasPrefix(url, vt.Prefix) {
			continue
		}
		path := strings.TrimPrefix(url, vt.Scheme)
		path = strings.TrimPrefix(path, "://")
		path = strings.TrimPrefix(path, "@")
		LogVerbose("Pinging path %s with scheme %s for vcs %s", path, vt.Scheme, vt.VCS.Name)
		if err := vt.VCS.Ping(vt.Scheme, path); err != nil {
			LogVerbose("Error pinging path %s with scheme %s", path, vt.Scheme)
			continue
		}
		return vt.VCS
	}
	return nil
}

func restoreWD(cwd string) {
	if err := os.Chdir(cwd); err != nil {
		log.Fatalf("Error restoring working directory: %s", err.Error())
	}

}

// PackageVCS wraps the underlying golang.org/x/tools/go/vcs to
// present the interface we need. It also implements the functionality
// necessary for SetRev to happen correctly. Multiple PackageVCS _can
// not be used concurrently_.
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

// GetRev does not work on remote VCS's and will always return a not
// implemented error.
func (pv *PackageVCS) GetRev() (string, error) {
	return "", errors.New("package VCS currently does not support GetRev")
}

// GetRoot will return pv.Repo.Root
func (pv *PackageVCS) GetRoot() string {
	return pv.Repo.Root
}

// GetSource returns the pv.Repo.Repo
func (pv *PackageVCS) GetSource() (string, error) {
	return pv.Repo.Repo, nil
}

// ErrorResolutionFailure will be returned if a RepoResolver could not
// resolve a VCS.
var ErrorResolutionFailure = errors.New("discovery failed")

// RepoResolver provides the mechanisms for resolving a VCS from an
// importpath and sourceUrl.
type RepoResolver interface {
	ResolveRepo(importPath string, dep *CanticleDependency) (VCS, error)
}

// DefaultRepoResolver attempts to resolve a repo using the go
// vcs.RepoRootForImportPath semantics and guessing logic.
type DefaultRepoResolver struct {
	Gopath string
}

// TrimPathToRoot will take import path github.comcast.com/x/tools/go/vcs
// and root golang.org/x/tools and create github.comcast.com/x/tools.
func TrimPathToRoot(importPath, root string) (string, error) {
	pathParts := strings.Split(importPath, "/")
	rootParts := strings.Split(root, "/")

	if len(pathParts) < len(rootParts) {
		return "", fmt.Errorf("path %s does not contain enough prefix for path %s", importPath, root)
	}
	return path.Join(pathParts[0:len(rootParts)]...), nil
}

// ResolveRepo on a default reporesolver is effectively go get wraped
// to use the url string.
func (dr *DefaultRepoResolver) ResolveRepo(importPath string, dep *CanticleDependency) (VCS, error) {
	// We guess our vcs based off our url path if present
	resolvePath := importPath

	LogVerbose("Attempting to use go get vcs for url: %s", resolvePath)
	vcs.Verbose = Verbose
	repo, err := vcs.RepoRootForImportPath(resolvePath, true)
	if err != nil {
		LogVerbose("Failed creating VCS for url: %s, err: %s", resolvePath, err.Error())
		return nil, err
	}

	// If we found something return non nil
	repo.Root, err = TrimPathToRoot(importPath, repo.Root)
	if err != nil {
		LogVerbose("Failed creating VCS for url: %s, err: %s", resolvePath, err.Error())
		return nil, err
	}
	v := &PackageVCS{Repo: repo, Gopath: dr.Gopath}
	LogVerbose("Created VCS for url: %s", resolvePath)
	return v, nil
}

// RemoteRepoResolver attempts to resolve a repo using the internal
// guessing logic for Canticle.
type RemoteRepoResolver struct {
	Gopath string
}

// ResolveRepo on the remoterepo resolver uses our own GuessVCS
// method. It mostly looks at protocol cues like svn:// and git@.
func (rr *RemoteRepoResolver) ResolveRepo(importPath string, dep *CanticleDependency) (VCS, error) {
	resolvePath := importPath
	if dep != nil && dep.SourcePath != "" {
		resolvePath = dep.SourcePath
	}
	// Attempt our internal guessing logic first
	LogVerbose("Attempting to use default resolver for url: %s", resolvePath)
	v := GuessVCS(resolvePath)
	if v == nil {
		return nil, ErrorResolutionFailure
	}

	root := dep.Root
	if root == "" {
		root = importPath
	}
	pv := &PackageVCS{
		Repo: &vcs.RepoRoot{
			VCS:  v,
			Repo: resolvePath,
			Root: root,
		},
		Gopath: rr.Gopath,
	}
	return pv, nil
}

// LocalRepoResolver will attempt to find local copies of a repo in
// LocalPath (treating it like a gopath) and provide VCS systems for
// updating them in RemotePath (also treaded like a gopath).
type LocalRepoResolver struct {
	LocalPath string
}

// ResolveRepo on a local resolver may return an error if:
// *  The local package is not present (no directory) in LocalPath
// *  The local "package" is a file in localpath
// *  There was an error stating the directory for the localPkg
func (lr *LocalRepoResolver) ResolveRepo(fullPath string, dep *CanticleDependency) (VCS, error) {
	LogVerbose("Finding local vcs for path: %s\n", fullPath)
	s, err := os.Stat(fullPath)
	switch {
	case err != nil:
		LogVerbose("Error stating local copy of package: %s %s\n", fullPath, err.Error())
		return nil, err
	case s != nil && s.IsDir():
		cmd, root, err := vcs.FromDir(fullPath, lr.LocalPath)
		if err != nil {
			LogVerbose("Error with local vcs: %s", err.Error())
			return nil, err
		}
		root, _ = PackageName(lr.LocalPath, path.Join(lr.LocalPath, root))
		v := NewLocalVCS(root, root, lr.LocalPath, cmd)
		LogVerbose("Created vcs for local pkg: %+v", v)
		return v, nil
	default:
		LogVerbose("Could not resolve local vcs for package: %s", fullPath)
		return nil, ErrorResolutionFailure
	}
}

// CompositeRepoResolver calls the repos in resolvers in order,
// discarding errors and returning the first VCS found.
type CompositeRepoResolver struct {
	Resolvers []RepoResolver
}

// ResolveRepo for the composite attempts its sub Resolvers in order
// ignoring any errors. If all resolvers fail ErrorResolutionFailure
// will be returned.
func (cr *CompositeRepoResolver) ResolveRepo(importPath string, dep *CanticleDependency) (VCS, error) {
	for _, r := range cr.Resolvers {
		vcs, err := r.ResolveRepo(importPath, dep)
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
// resolutions and will not attempt the same resolution twice.
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

// ResolveRepo on a MemoizedRepoResolver will cache the results of its
// child resolver.
func (mr *MemoizedRepoResolver) ResolveRepo(importPath string, dep *CanticleDependency) (VCS, error) {
	r := mr.resolvedPaths[importPath]
	if r != nil {
		return r.v, r.err
	}

	v, err := mr.resolver.ResolveRepo(importPath, dep)
	mr.resolvedPaths[importPath] = &resolve{v, err}
	return v, err
}
