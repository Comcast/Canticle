package canticles

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestEnvGoPath(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Could not get wd %s", err.Error())
	}
	defer os.Chdir(wd)
	path, err := EnvGoPath()
	if err != nil {
		t.Errorf("Error fetching obviously good path gopath %s", err.Error())
	}
	if path == "" {
		t.Errorf("Error fetching obviously good path got empty string")
	}
	testHome, err := ioutil.TempDir("", "cant-test-src")
	if err != nil {
		t.Fatalf("Error creating tempdir: %s", err.Error())
	}
	defer os.RemoveAll(testHome)

	if err := os.MkdirAll(filepath.Join(testHome, "src", "github.com", "comcast", "cant"), 0755); err != nil {
		t.Fatalf("Error creating tempdir sub folders: %s", err.Error())
	}
	if err := os.Chdir(testHome); err != nil {
		t.Fatalf("Could not chdir to created tmpdir: %s", err.Error())
	}

	if err := os.Setenv("GOPATH", testHome); err != nil {
		t.Fatalf("Could not set gopath: %s", err.Error())
	}
	path, err = EnvGoPath()
	if err != nil {
		t.Errorf("Expected no error when getting envgopath with a valid gopath, got %s", err.Error())
	}
	if path != testHome {
		t.Errorf("Expected path %s with a valid gopath, got %s", testHome, path)
	}

	if err := os.Setenv("GOPATH", ""); err != nil {
		t.Fatalf("Could not set gopath: %s", err.Error())
	}
	if err := os.Chdir(filepath.Join(testHome, "src")); err != nil {
		t.Fatalf("Could not chdir to created tmpdir: %s", err.Error())
	}
	path, err = EnvGoPath()
	if err != nil {
		t.Errorf("Expected no error when getting envgopath in a GB workspace, got %s", err.Error())
	}
	if path != testHome {
		t.Errorf("Expected path %s with a valid GB workspace, got %s", testHome, path)
	}

	// Create a "nested" gb workspace
	if err := os.MkdirAll(filepath.Join(testHome, "src", "nested", "src", "github.com", "comcast", "cant"), 0755); err != nil {
		t.Fatalf("Error creating tempdir sub folders: %s", err.Error())
	}
	wspace := filepath.Join(testHome, "src", "nested")
	if err := os.Chdir(filepath.Join(wspace, "src")); err != nil {
		t.Fatalf("Could not chdir to created tmpdir: %s", err.Error())
	}
	path, err = EnvGoPath()
	if err != nil {
		t.Errorf("Expected no error when getting envgopath in a GB workspace, got %s", err.Error())
	}
	if path != wspace {
		t.Errorf("Expected path %s with a valid GB workspace, got %s", testHome, path)
	}

	// Try an invalid spot
	if err := os.Chdir(os.TempDir()); err != nil {
		t.Fatalf("Could not chdir to created tmpdir: %s", err.Error())
	}
	path, err = EnvGoPath()
	if err == nil {
		t.Errorf("Expected an error when getting envgopath in an valid workspace, got")
	}
}
