package canticles

/*
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

func (tr *TestResolver) ResolveRepo(importPath string, dep *Dependency) (VCS, error) {
	r := tr.ResolvePaths[importPath]
	return r.V, r.Err
}
*/
