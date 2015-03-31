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
	cdr := &DepReader{}

	// Happy path
	deps, err := cdr.ReadCanticleDependencies("github.comcast.com/viper-cog/cant", os.ExpandEnv("$GOPATH"))
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
	deps, err = cdr.ReadCanticleDependencies("github.comcast.com/viper-cog/nothere", os.ExpandEnv("$GOPATH"))
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
	cdr := &DepReader{}

	// Happy path
	deps, err := cdr.ReadRemoteDependencies("github.comcast.com/viper-cog/cant", os.ExpandEnv("$GOPATH"))
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
	deps, err = cdr.ReadRemoteDependencies("github.comcast.com/viper-cog/nothere", os.ExpandEnv("$GOPATH"))
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
	cdr := &DepReader{}

	// Happy path
	deps, err := cdr.ReadDependencies("github.comcast.com/viper-cog/cant", os.ExpandEnv("$GOPATH"))
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
	deps, err = cdr.ReadDependencies("github.comcast.com/viper-cog/nothere", os.ExpandEnv("$GOPATH"))
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
	deps, err = cdr.ReadDependencies("github.comcast.com/viper-cog/cant/canticle", os.ExpandEnv("$GOPATH"))
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
	deps, err = cdr.ReadDependencies("github.comcast.com/viper-cog/cant", testHome)
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

type TestDepRead struct {
	Deps Dependencies
	Err  error
}

type TestDepReader struct {
	PackageDeps map[string]TestDepRead
}

func (tdr *TestDepReader) ReadDependencies(p, gopath string) (Dependencies, error) {
	r := tdr.PackageDeps[p]
	return r.Deps, r.Err
}

var CycledReader = &TestDepReader{
	PackageDeps: map[string]TestDepRead{
		"testpkg": TestDepRead{
			Deps: map[string]*Dependency{
				"dep1": &Dependency{
					ImportPath: "dep1",
					Revision:   "a",
				},
				"dep2": &Dependency{
					ImportPath: "dep2",
					Revision:   "b",
				},
			},
		},
		"dep1": TestDepRead{
			Deps: map[string]*Dependency{
				"dep1": &Dependency{
					ImportPath: "dep2",
				},
			},
		},
		"dep2": TestDepRead{
			Deps: map[string]*Dependency{
				"dep1": &Dependency{
					ImportPath: "dep1",
				},
			},
		},
	},
}

var CycleReaderDeps = map[string]*Dependency{
	"dep1": &Dependency{
		ImportPath: "dep1",
		Revision:   "a",
	},
	"dep2": &Dependency{
		ImportPath: "dep2",
		Revision:   "b",
	},
}

var NormalReader = &TestDepReader{
	PackageDeps: map[string]TestDepRead{
		"testpkg": TestDepRead{
			Deps: map[string]*Dependency{
				"dep1": &Dependency{
					ImportPath: "dep1",
					Revision:   "a",
				},
				"dep2": &Dependency{
					ImportPath: "dep2",
					Revision:   "b",
				},
			},
		},
		"dep1": TestDepRead{
			Deps: map[string]*Dependency{
				"dep1": &Dependency{
					ImportPath: "dep2",
				},
			},
		},
		"dep2": TestDepRead{
			Deps: map[string]*Dependency{},
		},
	},
}

var NormalReaderDeps = map[string]*Dependency{
	"dep1": &Dependency{
		ImportPath: "dep1",
		Revision:   "a",
	},
	"dep2": &Dependency{
		ImportPath: "dep2",
		Revision:   "b",
	},
}

var ChildCantReader = &TestDepReader{
	PackageDeps: map[string]TestDepRead{
		"testpkg": TestDepRead{
			Deps: map[string]*Dependency{
				"dep1": &Dependency{
					ImportPath: "dep1",
					Revision:   "a",
				},
				"dep2": &Dependency{
					ImportPath: "dep2",
					Revision:   "b",
				},
			},
		},
		"dep1": TestDepRead{
			Deps: map[string]*Dependency{
				"dep3": &Dependency{
					ImportPath: "dep3",
					Revision:   "d",
				},
			},
		},
		"dep2": TestDepRead{
			Deps: map[string]*Dependency{},
		},
		"dep3": TestDepRead{
			Deps: map[string]*Dependency{},
		},
	},
}

var ChildCantReaderDeps = map[string]*Dependency{
	"dep1": &Dependency{
		ImportPath: "dep1",
		Revision:   "a",
	},
	"dep2": &Dependency{
		ImportPath: "dep2",
		Revision:   "b",
	},
	"dep3": &Dependency{
		ImportPath: "dep3",
		Revision:   "d",
	},
}

var ErroredReader = &TestDepReader{
	PackageDeps: map[string]TestDepRead{
		"testpkg": TestDepRead{
			Deps: map[string]*Dependency{
				"dep1": &Dependency{
					ImportPath: "dep1",
					Revision:   "a",
				},
				"dep2": &Dependency{
					ImportPath: "dep2",
					Revision:   "b",
				},
			},
		},
		"dep1": TestDepRead{
			Deps: map[string]*Dependency{
				"dep3": &Dependency{
					ImportPath: "dep3",
					Revision:   "d",
				},
			},
		},
		"dep2": TestDepRead{
			Deps: map[string]*Dependency{},
		},
		"dep3": TestDepRead{
			Err: errors.New("Error package"),
		},
	},
}

func CheckDeps(t *testing.T, logPrefix string, expected, got Dependencies) {
	for k, v := range expected {
		if !reflect.DeepEqual(v, got.Dependency(k)) {
			t.Errorf("%s expected dep: %+v != got %+v", logPrefix, v, got[k])
		}
	}
}

func TestLoadAllDependencies(t *testing.T) {
	testHome := "test"
	// Run a test with a reader with cycled deps, make sure we don't infinite loop
	resolver := &TestResolver{}
	cdl := NewDependencyLoader(resolver, NormalReader, testHome)
	deps, err := cdl.LoadAllPackageDependencies("testpkg")
	if err != nil {
		t.Errorf("Error loading valid pkg %s", err.Error())
	}
	if len(deps) == 0 {
		t.Errorf("No valid deps for testpkg")
	}
	CheckDeps(t, "", NormalReaderDeps, deps)

	// Run a test with a reader with cycled deps, make sure we don't infinite loop
	resolver = &TestResolver{}
	cdl = NewDependencyLoader(resolver, CycledReader, testHome)
	deps, err = cdl.LoadAllPackageDependencies("testpkg")
	if err != nil {
		t.Errorf("Error loading valid pkg %s", err.Error())
	}
	if len(deps) == 0 {
		t.Errorf("No valid deps for testpkg")
	}
	CheckDeps(t, "", CycleReaderDeps, deps)

	// Run a test with child canticle revisions
	resolver = &TestResolver{}
	cdl = NewDependencyLoader(resolver, ChildCantReader, testHome)
	deps, err = cdl.LoadAllPackageDependencies("testpkg")
	if err != nil {
		t.Errorf("Error loading valid pkg %s", err.Error())
	}
	if len(deps) == 0 {
		t.Errorf("No valid deps for testpkg")
	}
	CheckDeps(t, "ChildCantReaderDeps", ChildCantReaderDeps, deps)

	// Run a test with a non loadable package
	resolver = &TestResolver{}
	cdl = NewDependencyLoader(resolver, ErroredReader, testHome)
	deps, err = cdl.LoadAllPackageDependencies("testpkg")
	if err == nil {
		t.Errorf("Error loading invvalid pkg %s", err.Error())
	}
	if len(deps) != 0 {
		t.Errorf("Valid deps for testpkg with error")
	}
}

func TestUpdateRepo(t *testing.T) {
	resolver := &TestResolver{}
	dl := NewDependencyLoader(resolver, NormalReader, "")
	if err := dl.UpdateRepo(&Dependency{Revision: ""}); err != nil {
		t.Errorf("UpdateRepo for nil revision returned err: %s", err.Error())
	}

	resolver = &TestResolver{
		ResolvePaths: map[string]*TestVCSResolve{
			"dep1": &TestVCSResolve{
				Err: errors.New("some error"),
			},
		},
	}
	dl = NewDependencyLoader(resolver, NormalReader, "")
	if err := dl.UpdateRepo(&Dependency{Revision: "a", ImportPath: "dep1"}); err == nil {
		t.Errorf("UpdateRepo for errored update returned nil err")
	}

	v := &TestVCS{}
	resolver = &TestResolver{
		ResolvePaths: map[string]*TestVCSResolve{
			"dep1": &TestVCSResolve{
				V: v,
			},
		},
	}
	dl = NewDependencyLoader(resolver, NormalReader, "")
	if err := dl.UpdateRepo(&Dependency{Revision: "a", ImportPath: "dep1"}); err != nil {
		t.Errorf("UpdateRepo for valid update returned err: %s", err.Error())
	}
	if v.Updated != 1 {
		t.Errorf("UpdateRepo updated %d times instead of %d times", v.Updated, 1)
	}
	if v.Rev != "a" {
		t.Errorf("UpdatedRepo send rev %s instead of %s", v.Rev, "a")
	}
}

func TestFetchRepo(t *testing.T) {
	resolver := &TestResolver{
		ResolvePaths: map[string]*TestVCSResolve{
			"dep1": &TestVCSResolve{
				Err: errors.New("some error"),
			},
		},
	}
	dl := NewDependencyLoader(resolver, NormalReader, "")
	if err := dl.FetchRepo(&Dependency{Revision: "a", ImportPath: "dep1"}); err == nil {
		t.Errorf("UpdateRepo for errored update returned nil err")
	}

	v := &TestVCS{}
	resolver = &TestResolver{
		ResolvePaths: map[string]*TestVCSResolve{
			"dep1": &TestVCSResolve{
				V: v,
			},
		},
	}
	dl = NewDependencyLoader(resolver, NormalReader, "")
	if err := dl.FetchRepo(&Dependency{Revision: "a", ImportPath: "dep1"}); err != nil {
		t.Errorf("UpdateRepo for valid update returned err: %s", err.Error())
	}
	if v.Created != 1 {
		t.Errorf("UpdateRepo updated %d times instead of %d times", v.Updated, 1)
	}
	if v.Rev != "a" {
		t.Errorf("UpdatedRepo send rev %s instead of %s", v.Rev, "a")
	}
}

func TestFetchUpdatePackage(t *testing.T) {
	testHome, err := ioutil.TempDir("", "cant-test")
	if err != nil {
		t.Fatalf("Error creating tempdir: %s", err.Error())
	}

	v := &TestVCS{}
	resolver := &TestResolver{
		ResolvePaths: map[string]*TestVCSResolve{
			"testpkg": &TestVCSResolve{
				V: v,
			},
		},
	}
	dl := NewDependencyLoader(resolver, NormalReader, testHome)
	if err := dl.FetchUpdatePackage(&Dependency{ImportPath: "testpkg"}); err != nil {
		t.Errorf("Error with fetchupdatepackage on create %s", err.Error())
	}
	if v.Created != 1 {
		t.Errorf("Fetchupdate package did not call vcs create")
	}

	name := path.Join(testHome, "src", "testpkg")
	os.MkdirAll(name, 0755)
	if err := dl.FetchUpdatePackage(&Dependency{ImportPath: "testpkg", Revision: "a"}); err != nil {
		t.Errorf("Error with fetchupdatepackage on update %s", err.Error())
	}
	if v.Updated != 1 {
		t.Errorf("Fetchupdate package did not call vcs update")
	}

	os.RemoveAll(name)
	ioutil.WriteFile(name, []byte("test"), 0655)
	if err := dl.FetchUpdatePackage(&Dependency{ImportPath: "testpkg"}); err == nil {
		t.Errorf("No error from fetchupdatepackage when file already exists but is not dir")
	}
	os.RemoveAll(testHome)
}
