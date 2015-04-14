package canticle

import "testing"

func TestDependenciesMarshalJSON(t *testing.T) {
	deps := NewDependencies()
	deps.AddDependency(&Dependency{ImportPaths: []string{"testpkg"}, Revision: "testrev"})
	bytes, err := deps.MarshalJSON()
	expectedJSON := `[{"ImportPaths":["testpkg"],"Revision":"testrev"}]`
	if err != nil {
		t.Errorf("Error marshaling valid deps: %s", err.Error())
	}
	if string(bytes) != expectedJSON {
		t.Errorf("Marshaled json %s did not match expected %s", string(bytes), expectedJSON)
	}

}

func TestDependenciesUnmarshalJSON(t *testing.T) {
	json :=
		[]byte(`[
	{
		"ImportPaths": ["golang.org/x/tools/go/vcs"],
		"Revision": "e4a1c78f0f69fbde8bb74f5e9f4adb037a68d753",
                "Root": "golang.org/x/tools"
	}
]`)

	deps := NewDependencies()
	if err := deps.UnmarshalJSON(json); err != nil {
		t.Errorf("Error unmarshaling valid json: %s", err.Error())
	}
	if len(deps) != 1 {
		t.Errorf("Dependencies did had %d elements, expected %d", len(deps), 1)
	}

	badJSON :=
		[]byte(`{
	"ImportPath": "golang.org/x/tools/go/vcs",
	"Revision": "e4a1c78f0f69fbde8bb74f5e9f4adb037a68d753"
}`)
	if err := deps.UnmarshalJSON(badJSON); err == nil {
		t.Errorf("No error unmarshaling invalid json")
	}
}

func TestDependenciesAddDependency(t *testing.T) {
	deps := NewDependencies()
	d1 := &Dependency{ImportPaths: []string{"root/import"}, Root: "root", SourcePath: "source", Revision: "rev"}
	d2 := &Dependency{ImportPaths: []string{"root/import/child"}, Root: "root", SourcePath: "source", Revision: "rev"}

	if err := deps.AddDependency(d1); err != nil {
		t.Errorf("Error adding valid dep %v %s", d1, err.Error())
	}
	dep := deps.Dependency("root")
	if dep != d1 {
		t.Errorf("Dep added %v not equal expected %v", dep, d1)
	}

	if err := deps.AddDependency(d2); err != nil {
		t.Errorf("Error adding valid dep %v %s", d2, err.Error())
	}

	dep = deps.Dependency("root")
	if dep != d1 {
		t.Errorf("Dep added %v not equal expected %v", dep, d1)
	}
	if len(d1.ImportPaths) != 2 {
		t.Errorf("Expected dep addition to add importpath")
	}

	d2.Revision = "conflictingrev"
	if err := deps.AddDependency(d2); err == nil {
		t.Errorf("Conflicting revision did not cause err")
	}

	d2.Revision = d1.Revision
	d2.SourcePath = "conflict"
	if err := deps.AddDependency(d2); err == nil {
		t.Errorf("Conflicting source did not cause err")
	}
	d2.SourcePath = d1.SourcePath

	d2.Revision = ""
	if err := deps.AddDependency(d2); err != nil {
		t.Errorf("Blank revision caused err")
	}
	d2.Revision = d1.Revision

	d2.SourcePath = ""
	if err := deps.AddDependency(d2); err != nil {
		t.Errorf("Blank sourcepath caused err")
	}
}
