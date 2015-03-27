package canticle

import (
	"fmt"
	"strings"

	"golang.org/x/tools/go/vcs"
)

// Attempt to guess the VCS given a url. This mostly relies on the
// protocols like "ssh://" etc.
func GuessVCS(url string) (v *vcs.Cmd, repo, scheme string) {
	switch {
	case strings.HasPrefix(url, "https://github.com"):
		return vcs.ByCmd("git"), strings.TrimPrefix(url, "https://"), "https"
	case strings.HasPrefix(url, "git+ssh://"):
		return vcs.ByCmd("git"), strings.TrimPrefix(url, "git+ssh://"), "git+ssh"
	case strings.HasPrefix(url, "git://"), strings.HasPrefix(url, "git@"):
		return vcs.ByCmd("git"), url, "git+ssh"
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

// RepoRootForImportPathWithURL uses a URL and an import path to guess
// a correct location and vcs for given import path. If the URL does
// not return a VCS from GuessVCS then it falls back on
// vcs.RepoRootForImportPath.
func RepoRootForImportPathWithURL(importPath, url string) (*vcs.RepoRoot, error) {
	fmt.Println("Guessing vcs for url: ", url)
	guess, path, scheme := GuessVCS(url)
	fmt.Printf("Guessed vcs: %s scheme: %s path: %s ", guess, scheme, path)
	if guess != nil {
		guess.Scheme = []string{scheme}
		repo := &vcs.RepoRoot{
			VCS:  guess,
			Repo: path,
			Root: importPath,
		}
		return repo, nil
		/*		err := guess.Ping(scheme, path)
				if err == nil {
					guess.Scheme = []string{scheme}
					return repo, nil
				} else {
					fmt.Println("Ping vcs err: ", err.Error())
				}
		*/
	}

	repo, err := vcs.RepoRootForImportPath(importPath, true)
	if err != nil {
		return nil, err
	}
	if url != "" {
		repo.Repo = url
	}
	return repo, nil
}
