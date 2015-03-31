package canticle

import (
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

func TestLoadAllDependencies(t *testing.T) {
}
