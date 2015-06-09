package canticles

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func TestBuild(t *testing.T) {
	// Test whether we can build ourselves!
	b.PreferLocals = true
	dir, err := ioutil.TempDir("", "cant-test")
	if err != nil {
		t.Fatalf("Error creating temp dir: %s", err.Error())
	}
	defer os.RemoveAll(dir)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting working directory: %s", err.Error())
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Error changing to temp dir")
	}
	defer os.Chdir(cwd)
	if err := b.BuildPackage("github.comcast.com/viper-cog/cant"); err != nil {
		t.Errorf("Error building ourself: %s", err.Error())
	} else {

	}
	b.PreferLocals = false
	b.InSource = true
	if err := b.WriteVersion("github.comcast.com/viper-cog/cant/canticles"); err != nil {
		t.Errorf("Error generating version info for non main package: %s", err.Error())
	}
	b.InSource = true

	p := path.Join(BuildSource(EnvGoPath(), "github.comcast.com/viper-cog/cant/canticles"), "generatedbuildinfo.go")
	if _, err := os.Stat(p); err == nil {
		t.Errorf("Generatedbuildinfo.go for non main package")
	}
}
