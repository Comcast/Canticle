package canticle

import (
	"os"
	"testing"
)

func TestRepoDiscovery(t *testing.T) {
	rd := NewRepoDiscovery(os.ExpandEnv("$GOROOT"))

	// NOTE: UPDATE ME IF WE EVER MOVE THIS
	importPath := "github.comcast.com/viper-cog/cant"
	url := "git@github.comcast.com:viper-cog/cant.git"
	vcs, err := rd.ResolveRepo(importPath, url)
	if err != nil {
		t.Errorf("ResolveRepo returned error for our own repo: %s", err.Error())
	}
	if vcs == nil {
		t.Fatalf("ResolveRepo returned nil vcs for repo: %s %s", importPath, url)
	}

	v := vcs.(*PackageVCS)
	if v.Repo.Root != importPath {
		t.Errorf("ResolveRepo did not set correct importpath for repo got %s expected %s", v.Repo.Root, importPath)
	}

	expectedURL := "github.comcast.com:viper-cog/cant.git"
	if v.Repo.Repo != expectedURL {
		t.Errorf("ResolveRepo did not set correct repo for repo got %s expected %s", v.Repo.Repo, expectedURL)
	}

	if rd.resolvedVCS[importPath] == nil {
		t.Errorf("ResolveRepo did not cache repo resolution")
	}

	// Try a VCS resolution against someone supports go get syntax
	importPath = "golang.org/x/tools/go/vcs"
	vcs, err = rd.ResolveRepo(importPath, "")
	if err != nil {
		t.Errorf("ResolveRepo returned error for golang.org repo: %s", err.Error())
	}
	if vcs == nil {
		t.Fatalf("ResolveRepo returned nil vcs for repo: %s %s", importPath, url)
	}

	v = vcs.(*PackageVCS)
	if v.Repo.Root != importPath {
		t.Errorf("ResolveRepo did not set correct importpath for repo got %s expected %s", v.Repo.Root, importPath)
	}
	if v.Repo.Repo == "" {
		t.Errorf("ResolveRepo did not set any repo for repo %s", importPath)
	}
	if rd.resolvedVCS[importPath] == nil {
		t.Errorf("ResolveRepo did not cache repo resolution")
	}

	// Try a VCS resolution that just flat fails
	importPath = "nothere.comcast.com/viper-cog/cant"
	url = "git@nothere.comcast.com:viper-cog/cant.git"
	vcs, err = rd.ResolveRepo(importPath, url)
	if err == nil {
		t.Errorf("ResolveRepo returned no error for a package that does not exist")
	}
	if vcs != nil {
		t.Fatalf("ResolveRepo returned non nil vcs for repo: %s %s", importPath, url)
	}
}
