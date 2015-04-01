package canticle

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"testing"
)

func TestReadCanticleDependencies(t *testing.T) {
	dr := &DepReader{os.ExpandEnv("$GOPATH")}

	// Happy path
	deps, err := dr.ReadCanticleDependencies("github.comcast.com/viper-cog/cant")
	if err != nil {
		t.Errorf("Error reading valid Canticle file %s error: %s", "github.comcast.com/viper-cog/cant", err.Error())
	}
	if deps == nil {
		t.Errorf("ReadCanticleDependencies should never return nil deps")
		return
	}
	if len(deps) != 1 {
		t.Errorf("ReadCanticleDependencies returned unexpected number of deps")
	}

	expected := &Dependency{
		ImportPath: "golang.org/x/tools/go/vcs",
		Revision:   "e4a1c78f0f69fbde8bb74f5e9f4adb037a68d753",
	}
	if !reflect.DeepEqual(deps[expected.ImportPath], expected) {
		t.Errorf("ReadCanticleDependencies returned %+v expected %+v", deps[expected.ImportPath], expected)
	}

	// Not so happy path
	deps, err = dr.ReadCanticleDependencies("github.comcast.com/viper-cog/nothere")
	if err == nil {
		t.Errorf("ReadCanticleDependencies returned nil error loading invalid path")
	}
	if deps == nil {
		t.Errorf("ReadCanticleDependencies should never return nil deps")
	}
	if len(deps) != 0 {
		t.Errorf("ReadCanticleDependencies returned non zero length deps loading invalid path")
	}
}

func TestReadRemoteDependencies(t *testing.T) {
	dr := &DepReader{os.ExpandEnv("$GOPATH")}

	// Happy path
	deps, err := dr.ReadRemoteDependencies("github.comcast.com/viper-cog/cant")
	if err != nil {
		t.Errorf("Error reading remotes for valid package %s error: %s", "github.comcast.com/viper-cog/cant", err.Error())
	}
	if deps == nil {
		t.Errorf("ReadRemoteDependencies should never return nil deps")
		return
	}
	if len(deps) != 1 {
		t.Errorf("ReadRemoteDependencies read incorrect number of deps")
	}

	expected := "github.comcast.com/viper-cog/cant/canticle"
	if deps[0] != expected {
		t.Errorf("ReadCanticleDependencies returned %+v expected %+v", deps[0], expected)
	}

	// Not so happy path
	deps, err = dr.ReadRemoteDependencies("github.comcast.com/viper-cog/nothere")
	if err == nil {
		t.Errorf("ReadRemoteDependencies returned nil error loading invalid path")
	}
	if deps == nil {
		t.Errorf("ReadRemoteDependencies should never return nil deps")
	}
	if len(deps) != 0 {
		t.Errorf("ReadRemoteDependencies returned non zero length deps loading invalid path")
	}
}

func TestReadDependencies(t *testing.T) {
	dr := &DepReader{os.ExpandEnv("$GOPATH")}

	// Happy path
	deps, err := dr.ReadDependencies("github.comcast.com/viper-cog/cant")
	if err != nil {
		t.Errorf("Error reading remotes for valid package %s error: %s", "github.comcast.com/viper-cog/cant", err.Error())
	}
	if deps == nil {
		t.Errorf("ReadDependencies should never return nil deps")
		return
	}
	if len(deps) != 2 {
		t.Errorf("ReadDependencies read incorrect number of deps, got %d, expected %d", len(deps), 2)
	}
	expected := &Dependency{
		ImportPath: "golang.org/x/tools/go/vcs",
		Revision:   "e4a1c78f0f69fbde8bb74f5e9f4adb037a68d753",
	}
	if !reflect.DeepEqual(deps[expected.ImportPath], expected) {
		t.Errorf("ReadCanticleDependencies returned %+v expected %+v", deps[expected.ImportPath], expected)
	}
	expected = &Dependency{
		ImportPath: "github.comcast.com/viper-cog/cant/canticle",
	}
	if !reflect.DeepEqual(deps[expected.ImportPath], expected) {
		t.Errorf("ReadCanticleDependencies returned %+v expected %+v", deps[expected.ImportPath], expected)
	}

	// Not so happy path
	deps, err = dr.ReadDependencies("github.comcast.com/viper-cog/nothere")
	if err == nil {
		t.Errorf("ReadDependencies returned nil error loading invalid path")
	}
	if deps == nil {
		t.Errorf("ReadDependencies should never return nil deps")
	}
	if len(deps) != 0 {
		t.Errorf("ReadDependencies returned non zero length deps loading invalid path")
	}

	// NonCanticle path
	deps, err = dr.ReadDependencies("github.comcast.com/viper-cog/cant/canticle")
	if err != nil {
		t.Errorf("ReadDependencies returned nil error loading valid path")
	}
	if deps == nil {
		t.Errorf("ReadDependencies should never return nil deps")
	}
	if len(deps) != 1 {
		t.Errorf("ReadDependencies returned non zero length deps loading invalid path")
	}
	expected = &Dependency{
		ImportPath: "golang.org/x/tools/go/vcs",
	}
	if !reflect.DeepEqual(deps[expected.ImportPath], expected) {
		t.Errorf("ReadCanticleDependencies returned %+v expected %+v", deps[expected.ImportPath], expected)
	}

	// Overriding path
	// Make a temp dir to copy these files into
	var testHome string
	testHome, err = ioutil.TempDir("", "cant-test")

	// Copy the files over
	testDir := path.Join(testHome, "src", "github.comcast.com/viper-cog/cant")
	source := path.Join(os.ExpandEnv("$GOPATH"), "src", "github.comcast.com/viper-cog/cant")
	dc := NewDirCopier(source, testDir)
	if err := dc.Copy(); err != nil {
		fmt.Printf("Error copying files %s\n", err.Error())
	}

	// Test ourselves
	dr.Gopath = testHome
	deps, err = dr.ReadDependencies("github.comcast.com/viper-cog/cant")
	if err != nil {
		t.Errorf("Error reading remotes for valid package %s error: %s", "github.comcast.com/viper-cog/cant", err.Error())
	}
	if deps == nil {
		t.Errorf("ReadDependencies should never return nil deps")
		return
	}
	if len(deps) != 2 {
		t.Errorf("ReadDependencies read incorrect number of deps for override path, got %d, expected %d", len(deps), 2)
	}
	expected = &Dependency{
		ImportPath: "golang.org/x/tools/go/vcs",
		Revision:   "e4a1c78f0f69fbde8bb74f5e9f4adb037a68d753",
	}
	if !reflect.DeepEqual(deps[expected.ImportPath], expected) {
		t.Errorf("ReadDependencies returned %+v expected %+v", deps[expected.ImportPath], expected)
	}
	expected = &Dependency{
		ImportPath: "github.comcast.com/viper-cog/cant/canticle",
	}
	if !reflect.DeepEqual(deps[expected.ImportPath], expected) {
		t.Errorf("ReadDependencies returned %+v expected %+v", deps[expected.ImportPath], expected)
	}
	if err = os.RemoveAll(testHome); err != nil {
		fmt.Printf("Error deleting files %s\n", err.Error())
	}
}

type TestDepRead struct {
	Deps Dependencies
	Err  error
}

type TestDepReader struct {
	PackageDeps map[string]TestDepRead
}

func (tdr *TestDepReader) ReadDependencies(p string) (Dependencies, error) {
	r := tdr.PackageDeps[p]
	return r.Deps, r.Err
}

type WalkCall struct {
	Dep  *Dependency
	Errs []error
}

type TestWalker struct {
	calls     []WalkCall
	responses []error
}

func (tw *TestWalker) HandlePackage(dep *Dependency, errs []error) error {
	tw.calls = append(tw.calls, WalkCall{dep, errs})
	if len(tw.responses) > 0 {
		resp := tw.responses[0]
		tw.responses = tw.responses[1:]
		return resp
	}
	return nil
}

var NormalReader = &TestDepReader{
	map[string]TestDepRead{
		"testpkg": TestDepRead{
			map[string]*Dependency{
				"dep1": &Dependency{ImportPath: "dep1", Revision: "a"},
				"dep2": &Dependency{ImportPath: "dep2", Revision: "b"},
			},
			nil,
		},
		"dep1": TestDepRead{
			map[string]*Dependency{
				"dep1": &Dependency{ImportPath: "dep2"},
			},
			nil,
		},
		"dep2": TestDepRead{
			map[string]*Dependency{}, nil,
		},
	},
}

var NormalReaderResult = []WalkCall{
	{&Dependency{ImportPath: "testpkg"}, nil},
	{&Dependency{ImportPath: "dep1", Revision: "a"}, nil},
	{&Dependency{ImportPath: "dep2", Revision: "b"}, nil},
}

var HandlerErrorResult = []WalkCall{
	{&Dependency{ImportPath: "testpkg"}, nil},
}

var HandlerSkipResult = []WalkCall{
	{&Dependency{ImportPath: "testpkg"}, nil},
}

var CycledReader = &TestDepReader{
	map[string]TestDepRead{
		"testpkg": TestDepRead{
			map[string]*Dependency{
				"dep1": &Dependency{ImportPath: "dep1", Revision: "a"},
				"dep2": &Dependency{ImportPath: "dep2", Revision: "b"},
			},
			nil,
		},
		"dep1": TestDepRead{
			map[string]*Dependency{
				"dep1": &Dependency{ImportPath: "dep2"},
			},
			nil,
		},
		"dep2": TestDepRead{
			map[string]*Dependency{
				"dep1": &Dependency{ImportPath: "dep1"},
			},
			nil,
		},
	},
}

var CycleReaderResult = []WalkCall{
	{&Dependency{ImportPath: "testpkg"}, nil},
	{&Dependency{ImportPath: "dep1", Revision: "a"}, nil},
	{&Dependency{ImportPath: "dep2", Revision: "b"}, nil},
}

var testErr = errors.New("Test err")
var ChildErrorReader = &TestDepReader{
	map[string]TestDepRead{
		"testpkg": TestDepRead{
			map[string]*Dependency{
				"dep1": &Dependency{ImportPath: "dep1", Revision: "a"},
				"dep2": &Dependency{ImportPath: "dep2", Revision: "b"},
			},
			nil,
		},
		"dep1": TestDepRead{
			map[string]*Dependency{
				"dep3": &Dependency{ImportPath: "dep3", Revision: "d"},
			},
			nil,
		},
		"dep2": TestDepRead{
			map[string]*Dependency{}, nil,
		},
		"dep3": TestDepRead{
			map[string]*Dependency{}, testErr,
		},
	},
}

var ChildErrorReaderResult = []WalkCall{
	{&Dependency{ImportPath: "testpkg"}, nil},
	{&Dependency{ImportPath: "dep1", Revision: "a"}, nil},
	{&Dependency{ImportPath: "dep2", Revision: "b"}, nil},
}

func CheckResult(t *testing.T, logPrefix string, expected, got []WalkCall) {
	for i, v := range expected {
		if !reflect.DeepEqual(v.Dep, got[i].Dep) {
			t.Errorf("%s expected dep: %+v != got %+v", logPrefix, v.Dep, got[i].Dep)
		}
		if !reflect.DeepEqual(v.Errs, got[i].Errs) {
			t.Errorf("%s expected err: %+v != got %+v", logPrefix, v.Errs, got[i].Errs)
		}

	}
}

func TestTraversePackageDependencies(t *testing.T) {
	tw := &TestWalker{}
	// Run a test with a reader with normal deps
	dw := NewDependencyWalker(NormalReader.ReadDependencies, tw.HandlePackage)
	err := dw.TraversePackageDependencies("testpkg")
	if err != nil {
		t.Errorf("Error loading valid pkg %s", err.Error())
	}
	CheckResult(t, "NormalReader", NormalReaderResult, tw.calls)

	// Run a test with an error from our handler
	tw = &TestWalker{responses: []error{nil, testErr}}
	dw = NewDependencyWalker(NormalReader.ReadDependencies, tw.HandlePackage)
	err = dw.TraversePackageDependencies("testpkg")
	if err == nil {
		t.Errorf("Error not returned from hanlder")
	}
	CheckResult(t, "HandlerError", HandlerErrorResult, tw.calls)

	// Run a test with a skip error from our handler
	tw = &TestWalker{responses: []error{nil, ErrorSkip}}
	dw = NewDependencyWalker(NormalReader.ReadDependencies, tw.HandlePackage)
	err = dw.TraversePackageDependencies("testpkg")
	if err != nil {
		t.Errorf("Error returned when skip form hanlder %s", err.Error())
	}
	CheckResult(t, "HandlerSkip", HandlerSkipResult, tw.calls)

	// Run a test with a reader with cycled deps, make sure we don't infinite loop
	tw = &TestWalker{}
	dw = NewDependencyWalker(CycledReader.ReadDependencies, tw.HandlePackage)
	err = dw.TraversePackageDependencies("testpkg")
	if err != nil {
		t.Errorf("Error loading valid pkg %s", err.Error())
	}
	CheckResult(t, "CycledReader", CycleReaderResult, tw.calls)

	// Run a test with a non loadable package
	tw = &TestWalker{}
	dw = NewDependencyWalker(ChildErrorReader.ReadDependencies, tw.HandlePackage)
	err = dw.TraversePackageDependencies("testpkg")
	if err == nil {
		t.Errorf("Error loading invvalid pkg %s", err.Error())
	}
	CheckResult(t, "ChildErrorReader", ChildErrorReaderResult, tw.calls)
}

type TestVCS struct {
	Updated int
	Created int
	Err     error
	Rev     string
}

func (v *TestVCS) Create(rev string) error {
	v.Rev = rev
	v.Created++
	return v.Err
}

func (v *TestVCS) SetRev(rev string) error {
	v.Rev = rev
	v.Updated++
	return v.Err
}

type TestVCSResolve struct {
	V   VCS
	Err error
}

type TestResolver struct {
	ResolvePaths map[string]*TestVCSResolve
}

func (tr *TestResolver) ResolveRepo(importPath, url string) (VCS, error) {
	r := tr.ResolvePaths[importPath]
	return r.V, r.Err
}

func TestFetchUpdatePackage(t *testing.T) {
	testHome, err := ioutil.TempDir("", "cant-test")
	if err != nil {
		t.Fatalf("Error creating tempdir: %s", err.Error())
	}
	defer os.RemoveAll(testHome)

	v := &TestVCS{}
	resolver := &TestResolver{
		ResolvePaths: map[string]*TestVCSResolve{
			"testpkg": &TestVCSResolve{
				V: v,
			},
		},
	}
	dl := NewDependencyLoader(resolver.ResolveRepo, testHome)
	if err := dl.FetchUpdatePackage(&Dependency{ImportPath: "testpkg"}, []error{}); err != nil {
		t.Errorf("Error with fetchupdatepackage on create %s", err.Error())
	}
	if v.Created != 1 {
		t.Errorf("Fetchupdate package did not call vcs create")
	}

	dl = NewDependencyLoader(resolver.ResolveRepo, testHome)
	name := path.Join(testHome, "src", "testpkg")
	os.MkdirAll(name, 0755)
	if err := dl.FetchUpdatePackage(&Dependency{ImportPath: "testpkg", Revision: "a"}, []error{}); err != nil {
		t.Errorf("Error with fetchupdatepackage on update %s", err.Error())
	}
	if v.Updated != 1 {
		t.Errorf("Fetchupdate package did not call vcs update")
	}

	dl = NewDependencyLoader(resolver.ResolveRepo, testHome)
	dl.HaltOnError = false
	if err := dl.FetchUpdatePackage(&Dependency{ImportPath: "testpkg"}, []error{testErr}); err != nil {
		t.Errorf("Error from fetchupdatepackage when passed an error with HaltOnError false")
	}

	dl = NewDependencyLoader(resolver.ResolveRepo, testHome)
	os.RemoveAll(name)
	ioutil.WriteFile(name, []byte("test"), 0655)
	if err := dl.FetchUpdatePackage(&Dependency{ImportPath: "testpkg"}, []error{}); err == nil {
		t.Errorf("No error from fetchupdatepackage when file already exists but is not dir")
	}

	dl = NewDependencyLoader(resolver.ResolveRepo, testHome)
	if err := dl.FetchUpdatePackage(&Dependency{ImportPath: "testpkg"}, []error{testErr}); err == nil {
		t.Errorf("No error from fetchupdatepackage when passed an error with HaltOnError true")
	}

}
