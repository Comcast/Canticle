package canticles

import (
	"os"
	"reflect"
	"testing"
)

func TestPackageIsRemote(t *testing.T) {
	imp := "github.comcast.com/viper-cog/canticle"
	if !IsRemote(imp) {
		t.Errorf("Import %s not marked as remote", imp)
	}

	imp = "io"
	if IsRemote(imp) {
		t.Errorf("Import %s marked as remote", imp)
	}

}

func TestPackageRemoteImports(t *testing.T) {
	pkg := &Package{
		Imports: []string{
			"github.comcast.com/viper-cog/canticle/canticle",
			"github.comcast.com/viper-cog/canticle/test",
			"io",
			"fmt",
		},
		TestImports: []string{
			"testing",
			"github.comcast.com/viper-cog/assert",
		},
	}

	expected := []string{
		"github.comcast.com/viper-cog/canticle/canticle",
		"github.comcast.com/viper-cog/canticle/test",
	}
	imps := pkg.RemoteImports(false)
	if !reflect.DeepEqual(imps, expected) {
		t.Errorf("Package remote imports: %v != %v", expected, imps)
	}

	imps = pkg.RemoteImports(true)
	expected = []string{
		"github.comcast.com/viper-cog/canticle/canticle",
		"github.comcast.com/viper-cog/canticle/test",
		"github.comcast.com/viper-cog/assert",
	}
	if !reflect.DeepEqual(imps, expected) {
		t.Errorf("Package remote imports: %v != %v", expected, imps)
	}
}

func TestLoadPackage(t *testing.T) {
	pkgPath := "github.comcast.com/viper-cog/cant/canticles"
	pkg, err := LoadPackage(pkgPath, os.ExpandEnv("$GOPATH"))
	if err != nil {
		t.Errorf("Error %s loading package information for valid package", err.Error())
	}
	if pkg == nil {
		t.Errorf("Loaded pkg not as expected")
		return
	}

	if pkg.ImportPath != pkgPath {
		t.Errorf("Loaded incorrect package, got %s != %s", pkg.ImportPath, pkgPath)
	}

	pkgPath = "nothere.comcast.com/nothere"
	pkg, err = LoadPackage(pkgPath, os.ExpandEnv("$GOPATH"))
	if err == nil {
		t.Errorf("No error loading invalid package")
	}
	if pkg != nil {
		t.Errorf("Loaded pkg not as expected")
		return
	}

}
