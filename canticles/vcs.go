package canticles

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"sync"

	"golang.org/x/tools/go/vcs"
)

// TODO: We should just rip out the reliance on tools/vcs. Most of it
// is so non functional it is just a headache.

// A VCS has the ability to create and change the revision of a
// package. A VCS is generall resolved using a RepoDiscovery.
type VCS interface {
	Create(rev string) error
	SetRev(rev string) error
	GetRev() (string, error)
	GetBranch() (string, error)
	UpdateBranch(branch string) (updated bool, update string, err error)
	GetSource() (string, error)
	GetRoot() string
}

// GitAtVCS creates a VCS cmd that supports the "git@blah.com:" syntax
func GitAtVCS() *vcs.Cmd {
	v := &vcs.Cmd{}
	*v = *vcs.ByCmd("git")
	v.CreateCmd = "clone {repo} {dir}"
	v.PingCmd = "ls-remote {scheme}@{repo}"
	v.Scheme = []string{"git"}
	v.PingCmd = "ls-remote {scheme}@{repo}"
	return v
}

// A VCSCmd is used to run a VCS command for a repo
type VCSCmd struct {
	Name       string
	Cmd        string
	Args       []string
	ParseRegex *regexp.Regexp
}

// ExecWithArgs overriden from the default
func (vc *VCSCmd) ExecWithArgs(repo string, args []string) (string, error) {
	LogVerbose("Running command: %s %v in dir %s", vc.Cmd, args, repo)
	cmd := exec.Command(vc.Cmd, args...)
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

// Exec executes this command with its arguments and parses them using
// regexp. Return an error if the command generates an error or we can
// not parse the results.
func (vc *VCSCmd) Exec(repo string) (string, error) {
	return vc.ExecWithArgs(repo, vc.Args)
}

// ExecReplace replaces the value in this commands args with values
// from vals and executes the function.
func (vc *VCSCmd) ExecReplace(repo string, vals map[string]string) (string, error) {
	replacements := make([]string, 0, len(vals)*2)
	for k, v := range vals {
		replacements = append(replacements, k, v)
	}
	replacer := strings.NewReplacer(replacements...)
	args := make([]string, 0, len(vc.Args))
	for _, arg := range vc.Args {
		args = append(args, replacer.Replace(arg))
	}
	return vc.ExecWithArgs(repo, args)
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

	// GitBranchCmd is used to get the current branch (if present)
	GitBranchCmd = &VCSCmd{
		Name:       "Git",
		Cmd:        "git",
		Args:       []string{"symbolic-ref", "--short", "HEAD"},
		ParseRegex: regexp.MustCompile(`(.+)`),
	}
	// HgBranchCmd is used to get the current branch (if present)
	HgBranchCmd = &VCSCmd{
		Name:       "Mercurial",
		Cmd:        "hg",
		Args:       []string{"id", "-b"},
		ParseRegex: regexp.MustCompile(`(.+)`),
	}
	// SvnBranchCmd is used to get the current branch (if present)
	SvnBranchCmd = &VCSCmd{
		Name:       "Subversion",
		Cmd:        "svn",
		Args:       []string{"info"},
		ParseRegex: regexp.MustCompile(`^URL: (.+)$`),
	}
	// BzrBranchCmd is used to get the current branch (if present)
	BzrBranchCmd = &VCSCmd{
		Name:       "Bazaar",
		Cmd:        "bzr",
		Args:       []string{"version-info"},
		ParseRegex: regexp.MustCompile(`branch-nick: (.+)`),
	}
	// BranchCmds is a map of cmd (git, svn, etc.) to
	// the cmd to parse the current branch
	BranchCmds = map[string]*VCSCmd{
		GitBranchCmd.Name: GitBranchCmd,
		SvnBranchCmd.Name: SvnBranchCmd,
		HgBranchCmd.Name:  HgBranchCmd,
		BzrBranchCmd.Name: BzrBranchCmd,
	}
)

// An UpdateCMD is used to update a local copy of remote branches and
// tags. Not relevant for Bazaar and SVN.
var (
	// GitUpdateCmd is used to update local copy's of remote branches (if present)
	GitUpdateCmd = &VCSCmd{
		Name:       "Git",
		Cmd:        "git",
		Args:       []string{"fetch", "--all"},
		ParseRegex: regexp.MustCompile(`(.+)`),
	}
	// HgUpdateCmd is used used to update local copy's of remote branches (if present)
	HgUpdateCmd = &VCSCmd{
		Name:       "Mercurial",
		Cmd:        "hg",
		Args:       []string{"pull"},
		ParseRegex: regexp.MustCompile(`(.+)`),
	}
	// BranchCmds is a map of cmd (git, svn, etc.) to
	// the cmd to parse the current branch
	UpdateCmds = map[string]*VCSCmd{
		GitUpdateCmd.Name: GitUpdateCmd,
		HgUpdateCmd.Name:  HgUpdateCmd,
	}
)

// A TagSyncCmd is used to set the revision of a git repo to the specified tag or branch.
var (
	GitTagSyncCmd = &VCSCmd{
		Name:       "Git",
		Cmd:        "git",
		Args:       []string{"checkout", "{tag}"},
		ParseRegex: regexp.MustCompile(`(.+)`),
	}
	HgTagSyncCmd = &VCSCmd{
		Name:       "Mercurial",
		Cmd:        "hg",
		Args:       []string{"update", "-r", "{tag}"},
		ParseRegex: regexp.MustCompile(`(.+)`),
	}
	BzrTagSyncCmd = &VCSCmd{
		Name:       "Bazaar",
		Cmd:        "bzr",
		Args:       []string{"update", "-r", "{tag}"},
		ParseRegex: regexp.MustCompile(`(Updated to .+|Tree is up)$`),
	}
	SvnTagSyncCmd = &VCSCmd{
		Name:       "Subversion",
		Cmd:        "svn",
		Args:       []string{"update", "--accept", "postpone", "-r", "{tag}"},
		ParseRegex: regexp.MustCompile(`(Updated to .+|At revision)`),
	}
	TagSyncCmds = map[string]*VCSCmd{
		GitTagSyncCmd.Name: GitTagSyncCmd,
		HgTagSyncCmd.Name:  HgTagSyncCmd,
		BzrTagSyncCmd.Name: BzrTagSyncCmd,
		SvnTagSyncCmd.Name: SvnTagSyncCmd,
	}
)

// A BranchUpdateCmd is used to update a branch (assumed to be already
// checked out) against a remote source. These commands will fail if
// the git equivalent of a "fast forward merge" can not be completed.
// The svn and bzr commands are the same as the tagsync commands.
var (
	GitBranchUpdateCmd = &VCSCmd{
		Name:       "Git",
		Cmd:        "git",
		Args:       []string{"pull", "--ff-only", "origin", "{branch}"},
		ParseRegex: regexp.MustCompile(`(Already|Updating .+)`),
	}
	HgBranchUpdateCmd = &VCSCmd{
		Name:       "Mercurial",
		Cmd:        "hg",
		Args:       []string{"pull", "-u"},
		ParseRegex: regexp.MustCompile(`(added .+|no changes found)$`),
	}
	BranchUpdateCmds = map[string]*VCSCmd{
		GitBranchUpdateCmd.Name: GitBranchUpdateCmd,
		HgBranchUpdateCmd.Name:  HgBranchUpdateCmd,
		BzrTagSyncCmd.Name:      BzrTagSyncCmd,
		SvnTagSyncCmd.Name:      SvnTagSyncCmd,
	}
	BranchUpdatedRegexs = map[string]*regexp.Regexp{
		GitBranchUpdateCmd.Name: regexp.MustCompile(`(Updating .+)`),
		HgBranchUpdateCmd.Name:  regexp.MustCompile(`(added .+)`),
		BzrTagSyncCmd.Name:      regexp.MustCompile(`(Updated to .+)`),
		SvnTagSyncCmd.Name:      regexp.MustCompile(`(Updated to .+)`),
	}
)

func GetSvnBranches(path string) ([]string, error) {
	return nil, errors.New("Not implemented")
}

func GetGitBranches(path string) ([]string, error) {
	cmd := exec.Command("git", "show-ref")
	cmd.Dir = path
	result, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(result), "\n")
	var results []string
	for _, line := range lines {
		parts := strings.Split(line, " ")
		if len(parts) > 1 {
			refName := parts[1]
			switch {
			case strings.HasPrefix(refName, "refs/heads/"):
				results = append(results, strings.TrimPrefix(refName, "refs/heads/"))
			case strings.HasPrefix(refName, "refs/remotes/"):
				// refs/remotes/origin/<branchname>
				remoteRef := strings.SplitN(strings.TrimPrefix(refName, "refs/remotes/"), "/", 2)
				results = append(results, remoteRef[1])
			}
		}
	}
	return results, nil
}

func GetHgBranches(path string) ([]string, error) {
	return nil, errors.New("Not implemented")
}

func GetBzrBranches(path string) ([]string, error) {
	return nil, errors.New("Not implemented")
}

var BranchFuncs = map[string]func(string) ([]string, error){
	GitBranchCmd.Name: GetGitBranches,
	SvnBranchCmd.Name: GetSvnBranches,
	HgBranchCmd.Name:  GetHgBranches,
	BzrBranchCmd.Name: GetBzrBranches,
}

// A LocalVCS uses packages and version control systems available at a
// local srcpath to control a local destpath (it copies the files over).
type LocalVCS struct {
	Package            string
	Root               string
	SrcPath            string
	Cmd                *vcs.Cmd
	CurrentRevCmd      *VCSCmd        // CurrentRevCommand to check the current revision for sourcepath.
	RemoteCmd          *VCSCmd        // RemoteCmd to obtain the upstream (remote) for a repo
	BranchCmd          *VCSCmd        // BranchCmd to obtains the current branch if on one
	UpdateCmd          *VCSCmd        // UpdateCMD is used to pull remote updates but NOT update the local
	BranchUpdateCmd    *VCSCmd        // BranchUpdateCmd is used to update a local branch with a remote
	BranchUpdatedRegex *regexp.Regexp // The regex to examine if an update occured from a branch update cmd
	SyncCmd            *VCSCmd
	Branches           func(path string) ([]string, error)
}

// NewLocalVCS returns a a LocalVCS with CurrentRevCmd initialized
// from the cmd's name using RevCmds and RemoteCmd from RemoteCmds.
func NewLocalVCS(pkg, root, srcPath string, cmd *vcs.Cmd) *LocalVCS {
	return &LocalVCS{
		Package:            pkg,
		Root:               root,
		SrcPath:            srcPath,
		Cmd:                cmd,
		CurrentRevCmd:      RevCmds[cmd.Name],
		RemoteCmd:          RemoteCmds[cmd.Name],
		BranchCmd:          BranchCmds[cmd.Name],
		UpdateCmd:          UpdateCmds[cmd.Name],
		Branches:           BranchFuncs[cmd.Name],
		BranchUpdateCmd:    BranchUpdateCmds[cmd.Name],
		BranchUpdatedRegex: BranchUpdatedRegexs[cmd.Name],
		SyncCmd:            TagSyncCmds[cmd.Name],
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
	src := PackageSource(lv.SrcPath, lv.Root)
	// Update against remotes if we need too
	if lv.UpdateCmd != nil {
		if _, err := lv.UpdateCmd.Exec(src); err != nil {
			return err
		}
	}
	// For revisions we just want to check it out
	if err := lv.TagSync(rev); err != nil {
		return err
	}
	return nil
}

func (lv *LocalVCS) TagSync(rev string) error {
	LogVerbose("Tag sync to: %s", rev)
	if lv.SyncCmd == nil {
		return nil
	}
	_, err := lv.SyncCmd.ExecReplace(PackageSource(lv.SrcPath, lv.Root), map[string]string{"{tag}": rev})
	if err == nil {
		return nil
	}
	LogVerbose("Tag sync failed with err: %s", err.Error())
	return lv.Cmd.TagSync(PackageSource(lv.SrcPath, lv.Root), rev)
}

func (lv *LocalVCS) RevIsBranch(rev string) bool {
	branches, err := lv.Branches(PackageSource(lv.SrcPath, lv.Root))
	if err != nil {
		LogVerbose("Error getting branches %s", err.Error())
		return false
	}
	LogVerbose("Found branches %v", branches)
	for _, br := range branches {
		if rev == br {
			return true
		}
	}
	return false
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

// GetBranch on a LocalVCS will return the branch (if any) for the
// current local repo. If none GetBranch will return an error.
func (lv *LocalVCS) GetBranch() (string, error) {
	return lv.BranchCmd.Exec(PackageSource(lv.SrcPath, lv.Root))
}

// UpdateBranch will return true if the local branch was updated,
// false if not. Error will be non nil if an error occured during the
// udpate.
func (lv *LocalVCS) UpdateBranch(branch string) (updated bool, update string, err error) {
	if !lv.RevIsBranch(branch) {
		return false, fmt.Sprintf("rev %s is not a branch", branch), nil
	}
	res, err := lv.BranchUpdateCmd.ExecReplace(
		PackageSource(lv.SrcPath, lv.Root),
		map[string]string{"{branch}": branch},
	)
	if lv.BranchUpdatedRegex.Match([]byte(res)) {
		return true, res, err
	}
	return false, res, err
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
	{"https://", "https", vcs.ByCmd("git")}, // not so sure this is a good idea
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

// PackageVCS wraps the underlying golang.org/x/tools/go/vcs to
// present the interface we need. It also implements the functionality
// necessary for SetRev to happen correctly.
type PackageVCS struct {
	Repo   *vcs.RepoRoot
	Gopath string
}

// UpdateBranch will attempt to construct a local vcs and update that.
func (pv *PackageVCS) UpdateBranch(branch string) (updated bool, update string, err error) {
	lv := NewLocalVCS(pv.Repo.Root, pv.Repo.Root, pv.Gopath, pv.Repo.VCS)
	return lv.UpdateBranch(branch)
}

// Create clones the VCS into the location provided by Repo.Root
func (pv *PackageVCS) Create(rev string) error {
	v := pv.Repo.VCS
	dir := PackageSource(pv.Gopath, pv.Repo.Root)
	if err := v.Create(dir, pv.Repo.Repo); err != nil {
		return err
	}
	if rev == "" {
		return nil
	}
	return pv.SetRev(rev)
}

// SetRev changes the revision of the Repo.Root to the value
// provided. This also modifies the git based vcs to be able to deal
// with non named revisions (sigh).
func (pv *PackageVCS) SetRev(rev string) error {
	lv := NewLocalVCS(pv.Repo.Root, pv.Repo.Root, pv.Gopath, pv.Repo.VCS)
	return lv.TagSync(rev)
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

// GetBranch does not work on remote VCS for now and will return an
// error.
func (pv *PackageVCS) GetBranch() (string, error) {
	return "", errors.New("package VCS currently does not support GetBranch")
}

// A ResolutionFailureError contains status as to whether this is a resolution failure
// or of some other type
type ResolutionFailureError struct {
	Err error
	Pkg string
	VCS string
}

// A NewResolutionFailureError with the pkg and vcs passed in
func NewResolutionFailureError(pkg, vcs string) *ResolutionFailureError {
	return &ResolutionFailureError{
		Err: fmt.Errorf("pkg %s could not be resolved by vcs %s", pkg, vcs),
		Pkg: pkg,
		VCS: vcs,
	}
}

// Error message attached to this vcs error
func (re ResolutionFailureError) Error() string {
	return re.Err.Error()
}

// ResolutionFailureErr will return non nil if a RepoResolver could not
// resolve a VCS.
func ResolutionFailureErr(err error) *ResolutionFailureError {
	if err == nil {
		return nil
	}
	if re, ok := err.(*ResolutionFailureError); ok {
		return re
	}
	return nil
}

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
	resolvePath := getResolvePath(importPath)

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
	resolvePath := getResolvePath(importPath)
	if dep != nil && dep.SourcePath != "" {
		resolvePath = getResolvePath(dep.SourcePath)
	}
	// Attempt our internal guessing logic first
	LogVerbose("Attempting to use default resolver for url: %s", resolvePath)
	v := GuessVCS(resolvePath)
	if v == nil {
		return nil, NewResolutionFailureError(importPath, "remote")
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

func getResolvePath(importPath string) string {
	if strings.Contains(importPath, "/") {
		return importPath
	} else {
		return importPath + "/"
	}
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
func (lr *LocalRepoResolver) ResolveRepo(pkg string, dep *CanticleDependency) (VCS, error) {
	LogVerbose("Finding local vcs for package: %s\n", pkg)
	fullPath := PackageSource(lr.LocalPath, getResolvePath(pkg))
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
		return nil, NewResolutionFailureError(pkg, "local")
	}
}

// CompositeRepoResolver calls the repos in resolvers in order,
// discarding errors and returning the first VCS found.
type CompositeRepoResolver struct {
	Resolvers []RepoResolver
}

// ResolveRepo for the composite attempts its sub Resolvers in order
// ignoring any errors. If all resolvers fail a ResolutionFailureError
// will be returned.
func (cr *CompositeRepoResolver) ResolveRepo(importPath string, dep *CanticleDependency) (VCS, error) {
	for _, r := range cr.Resolvers {
		vcs, err := r.ResolveRepo(importPath, dep)
		if vcs != nil && err == nil {
			return vcs, nil
		}
	}
	return nil, NewResolutionFailureError(importPath, "composite")
}

type resolve struct {
	v   VCS
	err error
}

// MemoizedRepoResolver remembers the results of previously attempted
// resolutions and will not attempt the same resolution twice.
type MemoizedRepoResolver struct {
	sync.RWMutex
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
	mr.RLock()
	r := mr.resolvedPaths[importPath]
	mr.RUnlock()
	if r != nil {
		return r.v, r.err
	}

	v, err := mr.resolver.ResolveRepo(importPath, dep)
	mr.Lock()
	mr.resolvedPaths[importPath] = &resolve{v, err}
	mr.Unlock()
	return v, err
}
