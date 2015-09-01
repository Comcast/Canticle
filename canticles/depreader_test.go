package canticles

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"testing"
)

func TestCanticleDependencies(t *testing.T) {
	dr := &DepReader{os.ExpandEnv("$GOPATH")}

	// Happy path
	deps, err := dr.CanticleDependencies("github.comcast.com/viper-cog/cant")
	if err != nil {
		t.Errorf("Error reading valid Canticle file %s error: %s", "github.comcast.com/viper-cog/cant", err.Error())
	}
	if deps == nil {
		t.Errorf("ReadCanticleDependencies should never return nil deps")
		return
	}
	if len(deps) != 1 {
		t.Errorf("ReadCanticleDependencies returned unexpected number of deps %d", len(deps))
	}
	expected := &CanticleDependency{
		Revision:   "e4a1c78f0f69fbde8bb74f5e9f4adb037a68d753",
		SourcePath: "https://go.googlesource.com/tools",
		Root:       "golang.org/x/tools",
	}
	if !reflect.DeepEqual(expected, deps[0]) {
		t.Errorf("Error reading cant deps: %v != %v", expected, deps[0])
	}

	// Not so happy path
	deps, err = dr.CanticleDependencies("github.comcast.com/viper-cog/nothere")
	if err == nil {
		t.Errorf("ReadCanticleDependencies returned nil error loading invalid path")
	}
	if len(deps) != 0 {
		t.Errorf("ReadCanticleDependencies returned non zero length deps loading invalid path")
	}
}

/*
func TestReadAllDeps(t *testing.T) {
	dir, err := ioutil.TempDir("", "cant-test")
	if err != nil {
		t.Fatalf("Could not create tmp directory with err %s", err.Error())
	}
	defer os.Remove(dir)

	// Create our file hierarchy
	paths := []string{"cubicle", "sosicle", ".fascicle"}
	for _, s := range paths {
		p := path.Join(dir, "src", "canttest", s)
		if err := os.MkdirAll(p, 0755); err != nil {
			t.Fatalf("Could not create tmp directory with err %s", err.Error())
		}
	}

	// Add Canticle files
	tragical := &Dependency{
		Root:     "tragic.com/tragical",
		Revision: "Oedipus",
	}
	cubicle := &Dependency{
		Root:     "hell.com/cubicle",
		Revision: "hell",
	}
	fascicle := &Dependency{
		Root:     "elsewhere.com/fascicle",
		Revision: "a seperately published work",
	}
	f, err := os.Create(path.Join(dir, "src", "canttest", "Canticle"))
	if err != nil {
		t.Fatalf("Error creating files for test %s", err.Error())
	}
	defer f.Close()
	deps := NewDependencies()
	deps.AddDependency(tragical)
	err = json.NewEncoder(f).Encode(deps)
	if err != nil {
		t.Fatalf("Error writing files for test %s", err.Error())
	}

	f1, err := os.Create(path.Join(dir, "src", "canttest", "cubicle", "Canticle"))
	if err != nil {
		t.Fatalf("Error creating files for test %s", err.Error())
	}
	defer f1.Close()
	deps = NewDependencies()
	deps.AddDependency(cubicle)
	err = json.NewEncoder(f1).Encode(deps)
	if err != nil {
		t.Fatalf("Error writing files for test %s", err.Error())
	}

	f2, err := os.Create(path.Join(dir, "src", "canttest", ".fascicle", "Canticle"))
	if err != nil {
		t.Fatalf("Error creating files for test %s", err.Error())
	}
	defer f2.Close()
	deps = NewDependencies()
	deps.AddDependency(fascicle)
	err = json.NewEncoder(f2).Encode(deps)
	if err != nil {
		t.Fatalf("Error writing files for test %s", err.Error())
	}

	// Setup all complete, lets read all our Canticle deps
	dr := &DepReader{dir}

	// Happy path
	result, err := dr.ReadAllCantDeps("canttest")
	if err != nil {
		t.Errorf("Error reading valid Canticle files %s error: %s", "canttest", err.Error())
	}
	if result == nil {
		t.Errorf("ReadAllCantDeps should never return nil deps")
		return
	}
	if len(result) != 2 {
		t.Errorf("ReadAllCantDeps returned unexpected number of deps %d", len(result))
	}
}

func setupTestPath(t *testing.T) (string, error) {
	dir, err := ioutil.TempDir("", "cant-test")
	if err != nil {
		t.Fatalf("Could not create tmp directory with err %s", err.Error())
		return "", err
	}

	// Create our file hierarchy
	paths := []string{"test.com/cubicle", "test.com/cubicle/sosicle", "test.com/fascicle"}
	for _, s := range paths {
		p := PackageSource(dir, s)
		if err := os.MkdirAll(p, 0755); err != nil {
			t.Fatalf("Could not create tmp directory with err %s", err.Error())
			return "", err
		}
		f, err := os.Create(path.Join(p, "test.go"))
		if err != nil {
			t.Fatalf("Could not create tmp file with err %s", err.Error())
			return "", err
		}
		defer f.Close()

		s := "package " + filepath.Base(p) + "\nimport _ \"test.com/fascicle\""
		_, err = f.Write([]byte(s))
		if err != nil {
			t.Fatalf("Could not write to tmp file with err %s", err.Error())
			return "", err
		}
	}

	// Sub dir with no buildable go files
	p := PackageSource(dir, "test.com/cubicle/empty")
	if err := os.MkdirAll(PackageSource(dir, p), 0755); err != nil {
		t.Fatalf("Could not create tmp directory with err %s", err.Error())
		return "", err
	}

	return dir, nil

}

func TestReadAllRemoteDeps(t *testing.T) {
	dir, err := setupTestPath(t)
	if err != nil {
		return
	}
	//defer os.Remove(dir)
	// Setup all complete, lets read all our Canticle deps
	dr := &DepReader{dir}

	result, err := dr.ReadAllRemoteDependencies("test.com/cubicle")
	if err != nil {
		t.Errorf("Error reading valid pkg files %s error: %s", "test.com/cubicle", err.Error())
	}
	if result == nil {
		t.Errorf("ReadAllRemoteDeps should never return nil deps")
		return
	}
	if len(result) != 1 {
		t.Errorf("ReadAllCantDeps returned unexpected number of deps %d", len(result))
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

	expected := "github.comcast.com/viper-cog/cant/canticles"
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
*/
func TestReadDependencies(t *testing.T) {
	dr := &DepReader{os.ExpandEnv("$GOPATH")}

	// Happy path
	deps, err := dr.GoRemoteDependencies("github.comcast.com/viper-cog/cant")
	if err != nil {
		t.Errorf("Error reading remotes for valid package %s error: %s", "github.comcast.com/viper-cog/cant", err.Error())
	}
	if deps == nil {
		t.Errorf("ReadRemoteDependencies should never return nil deps")
		return
	}
	if len(deps) != 2 {
		t.Errorf("ReadRemoteDependencies read incorrect number of deps, got %d, expected %d", len(deps), 2)
	}
	expected := "github.comcast.com/viper-cog/cant/buildinfo"
	if expected != deps[0] {
		t.Errorf("ReadRemoteDependencies returned %+v expected %+v", deps[0], expected)
	}
	expected = "github.comcast.com/viper-cog/cant/canticles"
	if expected != deps[1] {
		t.Errorf("ReadRemoteDependencies returned %+v expected %+v", deps[1], expected)
	}

	// Not so happy path
	deps, err = dr.GoRemoteDependencies("github.comcast.com/viper-cog/nothere")
	if err == nil {
		t.Errorf("ReadRemoteDependencies returned nil error loading invalid path")
	}
	if deps == nil {
		t.Errorf("ReadRemoteDependencies should never return nil deps")
	}
	if len(deps) != 0 {
		t.Errorf("ReadRemoteDependencies returned non zero length deps loading invalid path")
	}

	// Overriding path
	// Make a temp dir to copy these files into
	var testHome string
	testHome, err = ioutil.TempDir("", "cant-test")
	if err != nil {
		t.Fatalf("Error creating test dir: %s", err.Error())
	}
	defer os.RemoveAll(testHome)

	// Copy the files over
	testDir := path.Join(testHome, "src", "github.comcast.com/viper-cog/cant")
	source := path.Join(os.ExpandEnv("$GOPATH"), "src", "github.comcast.com/viper-cog/cant")
	dc := NewDirCopier(source, testDir)
	if err := dc.Copy(); err != nil {
		fmt.Printf("Error copying files %s\n", err.Error())
	}

	// Test ourselves
	dr.Gopath = testHome
	deps, err = dr.GoRemoteDependencies("github.comcast.com/viper-cog/cant")
	if err != nil {
		t.Errorf("Error reading remotes for valid package %s error: %s", "github.comcast.com/viper-cog/cant", err.Error())
	}
	if deps == nil {
		t.Errorf("ReadRemoteDependencies should never return nil deps")
		return
	}
	if len(deps) != 2 {
		t.Errorf("ReadRemoteDependencies read incorrect number of deps, got %d, expected %d", len(deps), 2)
	}
	expected = "github.comcast.com/viper-cog/cant/buildinfo"
	if expected != deps[0] {
		t.Errorf("ReadRemoteDependencies returned %+v expected %+v", deps[0], expected)
	}
	expected = "github.comcast.com/viper-cog/cant/canticles"
	if expected != deps[1] {
		t.Errorf("ReadRemoteDependencies returned %+v expected %+v", deps[1], expected)
	}
}
