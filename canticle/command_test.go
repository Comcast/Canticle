package canticle

import "testing"

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
	pkgs := []string{"test.com/testpkg,git@test.com:testpkg", "test.com/testpkg2,git@test.com:testpkg2"}
	deps := ParseCmdLineDeps(pkgs)

	expectedDeps := []*Dependency{
		&Dependency{ImportPath: "test.com/testpkg", SourcePath: "git@test.com:testpkg"},
		&Dependency{ImportPath: "test.com/testpkg2", SourcePath: "git@test.com:testpkg2"},
	}
	for i, dep := range deps {
		if expectedDeps[i].ImportPath != dep.ImportPath {
			t.Errorf("Expected dep: %s not equal result %s", expectedDeps[i].ImportPath, dep.ImportPath)
		}
		if expectedDeps[i].SourcePath != dep.SourcePath {
			t.Errorf("Expected dep: %s not equal result %s", expectedDeps[i].SourcePath, dep.SourcePath)
		}

	}
}
