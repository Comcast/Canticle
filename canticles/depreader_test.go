package canticles

import (
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
		t.Errorf("ReadCanticleDependencies returned unexpected number of deps %d", len(deps))
	}
	expected := &Dependency{
		ImportPaths: []string{"golang.org/x/tools/go/vcs"},
		Revision:    "e4a1c78f0f69fbde8bb74f5e9f4adb037a68d753",
	}
	if !reflect.DeepEqual(expected.ImportPaths, deps["golang.org/x/tools"].ImportPaths) {
		t.Errorf("Error expected importpaths: %v != %v", expected.ImportPaths, deps["golang.org/x/tools"].ImportPaths)
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
	deps, err := dr.ReadRemoteDependencies("github.comcast.com/viper-cog/cant")
	if err != nil {
		t.Errorf("Error reading remotes for valid package %s error: %s", "github.comcast.com/viper-cog/cant", err.Error())
	}
	if deps == nil {
		t.Errorf("ReadRemoteDependencies should never return nil deps")
		return
	}
	if len(deps) != 1 {
		t.Errorf("ReadRemoteDependencies read incorrect number of deps, got %d, expected %d", len(deps), 2)
	}
	expected := "github.comcast.com/viper-cog/cant/canticle"
	if expected != deps[0] {
		t.Errorf("ReadRemoteDependencies returned %+v expected %+v", deps[0], expected)
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
	deps, err = dr.ReadRemoteDependencies("github.comcast.com/viper-cog/cant")
	if err != nil {
		t.Errorf("Error reading remotes for valid package %s error: %s", "github.comcast.com/viper-cog/cant", err.Error())
	}
	if deps == nil {
		t.Errorf("ReadRemoteDependencies should never return nil deps")
		return
	}
	if len(deps) != 1 {
		t.Errorf("ReadRemoteDependencies read incorrect number of deps, got %d, expected %d", len(deps), 2)
	}
	expected = "github.comcast.com/viper-cog/cant/canticle"
	if expected != deps[0] {
		t.Errorf("ReadRemoteDependencies returned %+v expected %+v", deps[0], expected)
	}
}
