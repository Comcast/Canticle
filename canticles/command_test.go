package canticles

import (
	"reflect"
	"testing"
)

func TestPackageName(t *testing.T) {
	pkg, err := PackageName("/home/go", "/home/go/src/test.com/testpkg")
	if err != nil {
		t.Errorf("Error getting valid package: %s", err.Error())
	}
	pname := "test.com/testpkg"
	if pkg != pname {
		t.Errorf("Expected package %s got %s", pkg, pname)
	}
}

func TestParseCmdLineDeps(t *testing.T) {
	pkgs := []string{"test.com/testpkg", "test.com/testpkg2"}
	args := ParseCmdLinePackages(pkgs)

	if !reflect.DeepEqual(pkgs, args) {
		t.Errorf("Expected packages %v != %v", pkgs, args)
	}
}
