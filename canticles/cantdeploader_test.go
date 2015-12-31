package canticles

import (
	"errors"
	"sync"
	"testing"
)

type testCantDepReader struct {
	pkg  string
	deps []*CanticleDependency
	err  error
}

func (dr *testCantDepReader) CanticleDependencies(pkg string) ([]*CanticleDependency, error) {
	dr.pkg = pkg
	return dr.deps, dr.err
}

type resolution struct {
	vcs *TestVCS
	err error
}

type testRepoRes struct {
	sync.Mutex
	resolutions map[string]resolution
	calls       map[string]bool
}

func newTestRepoRes(resolutions map[string]resolution) *testRepoRes {
	return &testRepoRes{
		resolutions: resolutions,
		calls:       make(map[string]bool, len(resolutions)),
	}
}

func (rr *testRepoRes) ResolveRepo(importPath string, dep *CanticleDependency) (VCS, error) {
	rr.Lock()
	defer rr.Unlock()
	rr.calls[dep.Root] = true
	res := rr.resolutions[dep.Root]
	return res.vcs, res.err
}

type test struct {
	name                string
	path                string
	reader              *testCantDepReader
	resolver            *testRepoRes
	update              bool
	expectedRead        string
	expectedResolutions []*CanticleDependency
	expectedErrors      int
}

var (
	testError = errors.New("test error")
	testPath  = "/home/rfliam/go/src/testpkg"
	testPkg   = "testpkg"
	cdeps     = []*CanticleDependency{
		&CanticleDependency{
			Root:     "test1",
			Revision: "test1",
		},
		&CanticleDependency{
			Root:     "test2",
			Revision: "test2",
		},
		&CanticleDependency{
			Root:     "test3",
			Revision: "test3",
		},
	}
	v1          = &TestVCS{}
	v2          = &TestVCS{Err: testError}
	v3          = &TestVCS{}
	resolutions = map[string]resolution{
		"test1": resolution{v1, nil},
		"test2": resolution{v2, nil},
		"test3": resolution{v3, testError},
	}
	v4           = &TestVCS{}
	v5           = &TestVCS{Err: testError}
	v6           = &TestVCS{}
	resolutions2 = map[string]resolution{
		"test1": resolution{v4, nil},
		"test2": resolution{v5, nil},
		"test3": resolution{v6, testError},
	}
	tests = []test{
		test{
			name:           "Read returns an error",
			path:           testPath,
			reader:         &testCantDepReader{err: testError},
			expectedRead:   testPkg,
			expectedErrors: 1,
		},
		test{
			name:           "Don't update",
			path:           testPath,
			reader:         &testCantDepReader{deps: cdeps},
			resolver:       newTestRepoRes(resolutions),
			expectedRead:   testPkg,
			expectedErrors: 2,
		},
		test{
			name:           "Update",
			path:           testPath,
			reader:         &testCantDepReader{deps: cdeps},
			resolver:       newTestRepoRes(resolutions2),
			expectedRead:   testPkg,
			expectedErrors: 2,
			update:         true,
		},
	}
)

func TestCantDepLoader(t *testing.T) {
	gopath := "/home/rfliam/go"
	for _, test := range tests {
		loader := &CanticleDepLoader{
			Reader:   test.reader,
			Resolver: test.resolver,
			Gopath:   gopath,
			Update:   test.update,
		}
		errs := loader.FetchPath(test.path)
		if len(errs) != test.expectedErrors {
			t.Errorf("test %s: Expected %s errors, got %v", test.name, test.expectedErrors, errs)
		}
		if test.expectedRead != test.reader.pkg {
			t.Errorf("test %s: Expected read on pkg %s, got %s", test.name, test.expectedRead, test.reader.pkg)
		}
		for _, dep := range test.reader.deps {
			checkResolutions(t, test, dep)
		}
	}
}

func checkResolutions(t *testing.T, test test, dep *CanticleDependency) {
	test.resolver.Lock()
	defer test.resolver.Unlock()
	if !test.resolver.calls[dep.Root] {
		t.Errorf("test %s: Expected resolution on dep %v but not found", test.name, dep.Root)
	}
	res := test.resolver.resolutions[dep.Root]
	if res.err != nil {
		return
	}
	if res.vcs.Created != 1 {
		t.Errorf("test %s: Expected vcs for dep %s to be created for test, but it was not", test.name, dep.Root)
	}
	if res.vcs.Rev != dep.Revision {
		t.Errorf("test %s: Expected vcs to be created with rev %s got %s", test.name, dep.Revision, res.vcs.Rev)
	}

}
