package canticle

import (
	"os"
	"path"
	"testing"
)

func TestDefaultRepoResolver(t *testing.T) {
	dr := &DefaultRepoResolver{os.ExpandEnv("$GOPATH")}
	// Try a VCS resolution against someone supports go get syntax
	importPath := "golang.org/x/tools/go/vcs"
	vcs, err := dr.ResolveRepo(importPath, "")
	if err != nil {
		t.Errorf("DefaultRepoResolver returned error for golang.org repo: %s", err.Error())
	}
	if vcs == nil {
		t.Fatalf("DefaultRepoResolver returned nil vcs for repo: %s", importPath)
	}

	v := vcs.(*PackageVCS)
	if v.Repo.Root != "golang.org/x/tools" {
		t.Errorf("DefaultRepoResolver did not set correct root for repo got %s expected %s", v.Repo.Root, "golang.org/x/tools")
	}
	if v.Repo.Repo == "" {
		t.Errorf("DefaultRepoResolver did not set any repo for repo %s", importPath)
	}
}

func TestRemoteRepoResolver(t *testing.T) {
	rr := &RemoteRepoResolver{os.ExpandEnv("$GOPATH")}

	// NOTE: UPDATE ME IF WE EVER MOVE THIS
	importPath := "github.comcast.com/viper-cog/cant"
	url := "git@github.comcast.com:viper-cog/cant.git"
	vcs, err := rr.ResolveRepo(importPath, url)
	if err != nil {
		t.Errorf("RemoteRepoResolver returned error for our own repo: %s", err.Error())
	}
	if vcs == nil {
		t.Fatalf("RemoteRepoResolverResolveRepo returned nil vcs for repo: %s %s", importPath, url)
	}
	v := vcs.(*PackageVCS)
	if v.Repo.Root != importPath {
		t.Errorf("RemoteRepoResolver did not set correct importpath for repo got %s expected %s", v.Repo.Root, importPath)
	}
	expectedURL := "github.comcast.com:viper-cog/cant.git"
	if v.Repo.Repo != expectedURL {
		t.Errorf("ResolveRepo did not set correct repo for repo got %s expected %s", v.Repo.Repo, expectedURL)
	}

	// Try a VCS resolution that just flat fails
	importPath = "nothere.comcast.com/viper-cog/cant"
	url = "git@nothere.comcast.com:viper-cog/cant.git"
	vcs, err = rr.ResolveRepo(importPath, url)
	if err == nil {
		t.Errorf("RemoteRepoResolver returned no error for a package that does not exist")
	}
	if vcs != nil {
		t.Errorf("RemoteRepoResolver returned non nil vcs for repo: %s %s", importPath, url)
	}
}

func TestLocalRepoResolver(t *testing.T) {
	lr := &LocalRepoResolver{
		LocalPath:  os.ExpandEnv("$GOPATH"),
		RemotePath: "/tmp/",
	}

	pkg := "github.comcast.com/viper-cog/cant"
	vcs, err := lr.ResolveRepo(pkg, "")
	if err != nil {
		t.Errorf("LocalRepoResolver returned error resolving our own package %s", err.Error())
	}

	if vcs == nil {
		t.Fatalf("LocalRepoResolver returned a nil VCS resolving our own package")
	}
	v := vcs.(*LocalVCS)
	expectedSrc := path.Join(lr.LocalPath, "src", pkg)
	if v.SrcPath != expectedSrc {
		t.Errorf("LocalRepoResolver set vcs srcpath to %s expected %s", v.SrcPath, expectedSrc)
	}
	expectedDest := path.Join(lr.RemotePath, "src", pkg)
	if v.DestPath != expectedDest {
		t.Errorf("LocalRepoResolver set vcs destpath to %s expected %s", v.SrcPath, expectedSrc)
	}
	if v.Cmd.Cmd != "git" {
		t.Errorf("LocalRepoResolver did not set correct vcs command %s expected %s", v.Cmd.Cmd, "git")
	}

	// Test dealing with a package whose vcs root != the importpath
	pkg = "golang.org/x/tools/go/vcs"
	vcs, err = lr.ResolveRepo(pkg, "")
	if err != nil {
		t.Errorf("LocalRepoResolver returned error resolving our own package %s", err.Error())
	}

	if vcs == nil {
		t.Fatalf("LocalRepoResolver returned a nil VCS resolving our own package")
	}
	v = vcs.(*LocalVCS)
	expectedSrc = path.Join(lr.LocalPath, "src", "golang.org/x/tools/")
	if v.SrcPath != expectedSrc {
		t.Errorf("LocalRepoResolver set vcs srcpath to %s expected %s", v.SrcPath, expectedSrc)
	}
	expectedDest = path.Join(lr.RemotePath, "src", "golang.org/x/tools/")
	if v.DestPath != expectedDest {
		t.Errorf("LocalRepoResolver set vcs destpath to %s expected %s", v.SrcPath, expectedSrc)
	}
	if v.Cmd.Cmd != "git" {
		t.Errorf("LocalRepoResolver did not set correct vcs command %s expected %s", v.Cmd.Cmd, "git")
	}

}

type testResolve struct {
	path string
	url  string
}

type testResolver struct {
	resolutions []testResolve
	response    []resolve
}

func (tr *testResolver) ResolveRepo(i, u string) (VCS, error) {
	tr.resolutions = append(tr.resolutions, testResolve{i, u})
	resp := tr.response[0]
	tr.response = tr.response[1:]
	return resp.v, resp.err
}

func TestCompositeRepoResolver(t *testing.T) {
	res := &TestVCS{}
	tr1 := &testResolver{response: []resolve{{nil, testErr}}}
	tr2 := &testResolver{response: []resolve{{res, nil}}}

	cr := &CompositeRepoResolver{[]RepoResolver{tr1, tr2}}

	importpath := "testi"
	url := "testu"
	v, err := cr.ResolveRepo(importpath, url)
	if err != nil {
		t.Errorf("CompositeRepoResolver returned error with valid resolve %s", err.Error())
	}
	if v != res {
		t.Errorf("CompositeRepoResolver returned incorrect vcs")
	}
	if tr1.resolutions[0].path != importpath {
		t.Errorf("CompositeRepoResolver tr1 bad import path")
	}
	if tr1.resolutions[0].url != url {
		t.Errorf("CompositeRepoResolver tr1 bad url")
	}
	if tr2.resolutions[0].path != importpath {
		t.Errorf("CompositeRepoResolver tr2 bad import path")
	}
	if tr2.resolutions[0].url != url {
		t.Errorf("CompositeRepoResolver tr2 bad url")
	}

	tr1 = &testResolver{response: []resolve{{nil, testErr}}}
	tr2 = &testResolver{response: []resolve{{nil, testErr}}}
	cr = &CompositeRepoResolver{[]RepoResolver{tr1, tr2}}
	v, err = cr.ResolveRepo(importpath, url)
	if err != ErrorResolutionFailure {
		t.Errorf("CompositeRepoResolver did not return resolution failure")
	}
}

func TestMemoizedRepoResolver(t *testing.T) {
	res := &TestVCS{}
	tr1 := &testResolver{response: []resolve{{res, nil}}}
	mr := NewMemoizedRepoResolver(tr1)
	importpath := "testi"
	url := "testu"
	v, err := mr.ResolveRepo(importpath, url)
	if err != nil {
		t.Errorf("MemoizedRepoResolver returned error %s", err.Error())
	}
	if v != res {
		t.Errorf("MemoizedRepoResolver returned wrong vcs")
	}
	if len(tr1.resolutions) != 1 {
		t.Errorf("MemoizedRepoResolver did not call tr1 only once")
	}

	v, err = mr.ResolveRepo(importpath, url)
	if err != nil {
		t.Errorf("MemoizedRepoResolver returned error %s", err.Error())
	}
	if v != res {
		t.Errorf("MemoizedRepoResolver returned wrong vcs")
	}
	if len(tr1.resolutions) != 1 {
		t.Errorf("MemoizedRepoResolver did not call tr1 only once")
	}
}

// TODO: Add coverage for LocalVCS and PackageVCS
