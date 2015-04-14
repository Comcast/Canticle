package canticle

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

type TestDepRead struct {
	Deps []string
	Err  error
}

type TestDepReader struct {
	PackageDeps map[string]TestDepRead
}

func (tdr *TestDepReader) ReadDependencies(p string) ([]string, error) {
	r := tdr.PackageDeps[p]
	return r.Deps, r.Err
}

type TestWalker struct {
	calls     []string
	responses []error
}

func (tw *TestWalker) HandlePackage(pkg string) error {
	tw.calls = append(tw.calls, pkg)
	if len(tw.responses) > 0 {
		resp := tw.responses[0]
		tw.responses = tw.responses[1:]
		return resp
	}
	return nil
}

var NormalReader = &TestDepReader{
	map[string]TestDepRead{
		"testpkg": TestDepRead{[]string{"dep1", "dep2"}, nil},
		"dep1":    TestDepRead{[]string{"dep2"}, nil},
		"dep2":    TestDepRead{},
	},
}

var NormalReaderResult = []string{"testpkg", "dep1", "dep2"}

var HandlerErrorResult = []string{"testpkg"}

var HandlerSkipResult = []string{"testpkg"}

var CycledReader = &TestDepReader{
	map[string]TestDepRead{
		"testpkg": TestDepRead{[]string{"dep1", "dep2"}, nil},
		"dep1":    TestDepRead{[]string{"dep2"}, nil},
		"dep2":    TestDepRead{[]string{"dep1"}, nil},
	},
}

var CycleReaderResult = []string{"testpkg", "dep1", "dep2"}

var ChildErrorReader = &TestDepReader{
	map[string]TestDepRead{
		"testpkg": TestDepRead{[]string{"dep1", "dep2"}, nil},
		"dep1":    TestDepRead{[]string{"dep3"}, nil},
		"dep2":    TestDepRead{},
		"dep3":    TestDepRead{[]string{}, errTest},
	},
}

var ChildErrorReaderResult = []string{"testpkg", "dep1", "dep2"}

func CheckResult(t *testing.T, logPrefix string, expected, got []string) {
	for i, v := range expected {
		if v != got[i] {
			t.Errorf("%s expected dep: %+v != got %+v", logPrefix, v, got[i])
		}
	}
}

func TestTraverseDependencies(t *testing.T) {
	tw := &TestWalker{}
	// Run a test with a reader with normal deps
	dw := NewDependencyWalker(NormalReader.ReadDependencies, tw.HandlePackage)
	err := dw.TraverseDependencies("testpkg")
	if err != nil {
		t.Errorf("Error loading valid pkg %s", err.Error())
	}
	CheckResult(t, "NormalReader", NormalReaderResult, tw.calls)

	// Run a test with an error from our handler
	tw = &TestWalker{responses: []error{nil, errTest}}
	dw = NewDependencyWalker(NormalReader.ReadDependencies, tw.HandlePackage)
	err = dw.TraverseDependencies("testpkg")
	if err == nil {
		t.Errorf("Error not returned from hanlder")
	}
	CheckResult(t, "HandlerError", HandlerErrorResult, tw.calls)

	// Run a test with a skip error from our handler
	tw = &TestWalker{responses: []error{nil, ErrorSkip}}
	dw = NewDependencyWalker(NormalReader.ReadDependencies, tw.HandlePackage)
	err = dw.TraverseDependencies("testpkg")
	if err != nil {
		t.Errorf("Error returned when skip form hanlder %s", err.Error())
	}
	CheckResult(t, "HandlerSkip", HandlerSkipResult, tw.calls)

	// Run a test with a reader with cycled deps, make sure we don't infinite loop
	tw = &TestWalker{}
	dw = NewDependencyWalker(CycledReader.ReadDependencies, tw.HandlePackage)
	err = dw.TraverseDependencies("testpkg")
	if err != nil {
		t.Errorf("Error loading valid pkg %s", err.Error())
	}
	CheckResult(t, "CycledReader", CycleReaderResult, tw.calls)

	// Run a test with a non loadable package
	tw = &TestWalker{}
	dw = NewDependencyWalker(ChildErrorReader.ReadDependencies, tw.HandlePackage)
	err = dw.TraverseDependencies("testpkg")
	if err == nil {
		t.Errorf("Error loading invvalid pkg %s", err.Error())
	}
	CheckResult(t, "ChildErrorReader", ChildErrorReaderResult, tw.calls)
}

type TestCantRead struct {
	Deps Dependencies
	Err  error
}

type TestCantReader struct {
	PackageDeps map[string]TestCantRead
}

func (tdr *TestCantReader) Read(p string) (Dependencies, error) {
	r := tdr.PackageDeps[p]
	return r.Deps, r.Err
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
	v2 := &TestVCS{}
	tr := &TestResolver{
		ResolvePaths: map[string]*TestVCSResolve{
			"testpkg": &TestVCSResolve{
				V: v,
			},
			"root/import": &TestVCSResolve{
				V: v2,
			},
		},
	}
	canticleDeps := NewDependencies()
	dep := &Dependency{ImportPaths: []string{"root/import"}, SourcePath: "source", Root: "root", Revision: "rev"}
	canticleDeps.AddDependency(dep)
	read := TestCantRead{Deps: canticleDeps, Err: nil}
	read2 := TestCantRead{Deps: nil, Err: os.ErrNotExist}
	testCantReader := &TestCantReader{map[string]TestCantRead{"testpkg": read, "root/import": read2}}

	dl := NewDependencyLoader(tr, testCantReader.Read, testHome)
	if err := dl.FetchUpdatePackage("testpkg"); err != nil {
		t.Errorf("Error with fetchupdatepackage on create %s", err.Error())
	}
	if v.Created != 1 {
		t.Errorf("Fetchupdate package did not call vcs create")
	}

	dl = NewDependencyLoader(tr, testCantReader.Read, testHome)
	name := PackageSource(testHome, "testpkg")
	os.MkdirAll(name, 0755)
	if err := dl.FetchUpdatePackage("testpkg"); err != nil {
		t.Errorf("Error with fetchupdatepackage on update %s", err.Error())
	}
	if v.Updated != 1 {
		t.Errorf("Fetchupdate package did not call vcs update")
	}
	if len(dl.FetchedDeps()) != 1 {
		t.Errorf("FetchUpdatePackage had %d instead of %d deps as expected", len(dl.FetchedDeps()), 1)
	}
	// A subsequent call to this for "import" should result in the same thing
	if err := dl.FetchUpdatePackage(dep.ImportPaths[0]); err != nil {
		t.Errorf("Error with fetchupdatepackage on update %s", err.Error())
	}
	if len(dl.FetchedDeps()) != 1 {
		t.Errorf("FetchUpdatePackage had %d instead of %d deps as expected", len(dl.FetchedDeps()), 1)
	}
	if v2.Created != 1 {
		t.Errorf("Fetchupdate package did not call vcs update")
	}
	if v2.Rev != dep.Revision {
		t.Errorf("FetchUpdatePackage called create with rev %s expected %s", v2.Rev, dep.Revision)
	}

	dl = NewDependencyLoader(tr, testCantReader.Read, testHome)
	os.RemoveAll(name)
	ioutil.WriteFile(name, []byte("test"), 0655)
	if err := dl.FetchUpdatePackage("testpkg"); err == nil {
		t.Errorf("No error from fetchupdatepackage when file already exists but is not dir")
	}

	dl = NewDependencyLoader(tr, testCantReader.Read, testHome)
	if err := dl.FetchUpdatePackage("testpkg"); err == nil {
		t.Errorf("No error from fetchupdatepackage when passed an error with HaltOnError true")
	}

}

func TestDependencySaver(t *testing.T) {
	testHome, err := ioutil.TempDir("", "cant-test")
	if err != nil {
		t.Fatalf("Error creating tempdir: %s", err.Error())
	}
	defer os.RemoveAll(testHome)

	v := &TestVCS{
		Rev:    "r1",
		Root:   "blah.com/root",
		Source: "git@blah.com:root.git",
	}
	v2 := &TestVCS{
		Rev:    "r1",
		Root:   "blah.com/root",
		Source: "git@blah.com:root.git",
	}
	childpkg := "blah.com/root/child"
	rootpkg := "blah.com/root"
	resolver := &TestResolver{
		ResolvePaths: map[string]*TestVCSResolve{
			childpkg: &TestVCSResolve{
				V: v,
			},
			rootpkg: &TestVCSResolve{
				V: v2,
			},
		},
	}

	ds := NewDependencySaver(resolver, testHome)
	if err := ds.SavePackageRevision(childpkg); err == nil {
		t.Errorf("Did not report err loading package with no file")
	}

	name := PackageSource(testHome, childpkg)
	fmt.Printf("Making dir %s\n", name)
	os.MkdirAll(name, 0755)

	if err := ds.SavePackageRevision(childpkg); err != nil {
		t.Errorf("Err loading valid package: %s", err.Error())
	}
	deps := ds.Dependencies()
	if len(deps) != 1 {
		t.Errorf("Err incorrect number of deps, expected 1 got %d", len(deps))
	}
	tp := deps[rootpkg]
	if tp == nil {
		t.Fatal("Err deps nil at testpkg")
	}
	if tp.Revision != v.Rev {
		t.Errorf("Expected testpkg revision to be %s, got %s", v.Rev, tp.Revision)
	}
	if tp.SourcePath != v.Source {
		t.Errorf("Expected source %s, got %s", v.Source, tp.SourcePath)
	}

	if err := ds.SavePackageRevision(rootpkg); err != nil {
		t.Errorf("Error loading valid package: %s", err)
	}
	deps = ds.Dependencies()
	if len(deps) != 1 {
		t.Errorf("Err incorrect number of deps, expected 1 got %d", len(deps))
	}
	tp = deps[rootpkg]
	if tp == nil {
		t.Fatal("Err deps nil at testpkg")
	}

	if err := ds.SavePackageRevision("testpkg"); err == nil {
		t.Errorf("Did not report err when passed err")
	}
}
