package canticles

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"testing"
)

var mainTemplate = `package main

import (
     "encoding/json"
     "fmt"
     "log"
     "./buildinfo"
)

func main() {
        b, err := json.Marshal(buildinfo.GetBuildInfo())
        if err != nil {
               log.Fatalf("Error marshaling: %s", err.Error())
        }
	fmt.Printf("BuildInfo: %s\n", string(b))
}
`

var deps = []*CanticleDependency{}

func TestBuildInfo(t *testing.T) {
	bi, err := NewBuildInfo("test", deps)
	if err != nil {
		t.Errorf("Error not nil obtaining information about our own package: %s", err.Error())
	}
	dir, err := ioutil.TempDir("", "cant-test")
	if err != nil {
		t.Fatalf("Error creating temp dir: %s", err.Error())
	}
	defer os.RemoveAll(dir)
	if err := ioutil.WriteFile(path.Join(dir, "main.go"), []byte(mainTemplate), 0644); err != nil {
		t.Fatalf("Error writing temp main: %s", err.Error())
	}

	if err := bi.WriteFiles(dir); err != nil {
		t.Errorf("Error writing buildinfo go file: %s", err.Error())
	}

	// Check that it compiles
	cmd := exec.Command("go", "build")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("Error generating built file, output: %s", string(output))
	}

	output, err = exec.Command(path.Join(dir, path.Base(dir))).CombinedOutput()
	if err != nil {
		t.Errorf("Error generating built file, output: %s", string(output))
	}
	t.Log(string(output))
}
