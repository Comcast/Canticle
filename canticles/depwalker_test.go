package canticles

import (
	"io/ioutil"
	"os"
	"path"
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

type TestVCSResolve struct {
	V   VCS
	Err error
}

type TestResolver struct {
	ResolvePaths map[string]*TestVCSResolve
}

func (tr *TestResolver) ResolveRepo(importPath string, dep *CanticleDependency) (VCS, error) {
	r := tr.ResolvePaths[importPath]
	return r.V, r.Err
}

type TestDependencyRead struct {
	Deps Dependencies
	Err  error
}

type TestDependencyReader struct {
	PackageDeps map[string]TestDependencyRead
}

func (tdr *TestDependencyReader) ReadDependencies(p string) (Dependencies, error) {
	r := tdr.PackageDeps[p]
	return r.Deps, r.Err
}

func TestDependencyLoader(t *testing.T) {
	// Create our
	testHome, err := ioutil.TempDir("", "cant-test")
	if err != nil {
		t.Fatalf("Error creating tempdir: %s", err.Error())
	}
	defer os.RemoveAll(testHome)
	if err := os.MkdirAll(path.Join(testHome, "src", "pkg1", "child"), 0755); err != nil {
		t.Fatal(err)
	}
	pkg1dep := map[string]*Dependency{
		"pkg1/child": NewDependency("pkg1/child"),
	}
	pkg1childdep := map[string]*Dependency{
		"pkg2/child": NewDependency("pkg2/child"),
	}
	deps := &TestDependencyReader{
		map[string]TestDependencyRead{
			"pkg1":       TestDependencyRead{pkg1dep, nil},
			"pkg1/child": TestDependencyRead{pkg1childdep, nil},
			"pkg2":       TestDependencyRead{NewDependencies(), nil},
			"pkg2/child": TestDependencyRead{NewDependencies(), nil},
		},
	}

	cdeps := []*CanticleDependency{
		&CanticleDependency{Root: "pkg1"},
		&CanticleDependency{Root: "pkg2"},
	}
	pkg1vcs := &TestVCS{}
	pkg2vcs := &TestVCS{}
	tr := &TestResolver{map[string]*TestVCSResolve{
		"pkg1":       &TestVCSResolve{pkg1vcs, nil},
		"pkg1/child": &TestVCSResolve{pkg1vcs, nil},
		"pkg2":       &TestVCSResolve{pkg2vcs, nil},
		"pkg2/child": &TestVCSResolve{pkg2vcs, nil},
	}}
	dl := NewDependencyLoader(tr, deps.ReadDependencies, cdeps, testHome)

	if err := dl.FetchUpdatePackage("pkg1"); err != nil {
		t.Errorf("Error fetching pkg1: %s", err.Error())
	}
	pkgImports, err := dl.PackageImports("pkg1")
	if err != nil {
		t.Errorf("Error getting imports for pkg1: %s", err.Error())
	}
	if pkgImports[0] != "pkg1/child" {
		t.Errorf("Expected pkg1 to have imports pkg1/child got: %s", pkgImports[0])
	}
	if pkg1vcs.Created != 0 {
		t.Errorf("Expected pkg1vcs to have no creates: %d", pkg1vcs.Created)
	}
	if err := dl.FetchUpdatePackage("pkg1/child"); err != nil {
		t.Errorf("Error fetching pkg1: %s", err.Error())
	}
	pkgImports, err = dl.PackageImports("pkg1/child")
	if err != nil {
		t.Errorf("Error getting imports for pkg1: %s", err.Error())
	}
	if pkgImports[0] != "pkg2/child" {
		t.Errorf("Expected pkg1 to have imports pkg2/child got: %s", pkgImports[0])
	}
	if pkg1vcs.Created != 0 {
		t.Errorf("Expected pkg1vcs to have no creates: %d", pkg1vcs.Created)
	}

	if err := dl.FetchUpdatePackage("pkg2/child"); err != nil {
		t.Errorf("Error fetching pkg2: %s", err.Error())
	}
	pkgImports, err = dl.PackageImports("pkg2/child")
	if err != nil {
		t.Errorf("Error getting imports for pkg2: %s", err.Error())
	}
	if len(pkgImports) != 0 {
		t.Errorf("Expected pkg2 to have no imports pkg2/child got: %d", len(pkgImports))
	}
	if pkg2vcs.Created != 1 {
		t.Errorf("Expected pkg2vcs to have 1 create: %d", pkg2vcs.Created)
	}

}
