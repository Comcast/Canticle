package canticle

import (
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type DirCopier struct {
	source, dest string
	CopyDot      bool
}

func NewDirCopier(source, dest string) *DirCopier {
	return &DirCopier{source, dest, false}
}

func (dc *DirCopier) Copy() error {
	return filepath.Walk(dc.source, dc.cp)
}

func (dc *DirCopier) cp(path string, f os.FileInfo, err error) error {
	if !dc.CopyDot && strings.HasPrefix(filepath.Base(path), ".") {
		if f.IsDir() {
			return filepath.SkipDir
		}
		return nil
	}
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(dc.source, path)
	if err != nil {
		return err
	}
	if f.IsDir() {
		dest := filepath.Join(dc.dest, rel)
		return os.MkdirAll(dest, f.Mode())
	}
	s, err := os.Open(path)
	if err != nil {
		return err
	}
	defer s.Close()

	dst := filepath.Join(dc.dest, rel)
	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer d.Close()
	if _, err := io.Copy(d, s); err != nil {
		return err
	}
	return nil
}

// PatchEnviroment changes an enviroment variable set to
// have a new key value
func PatchEnviroment(env []string, key, value string) []string {
	prefix := key + "="
	newValue := key + "=" + value
	for i, v := range env {
		if strings.HasPrefix(v, prefix) {
			env[i] = newValue
			return env
		}
	}
	return append(env, newValue)
}

// EnvGoPath returns the enviroments GOPATH variable
func EnvGoPath() string {
	return os.Getenv("GOPATH")
}

// PathIsChild will return true if the child path
// is a subfolder of parent.
func PathIsChild(parent, child string) bool {
	parentParts := strings.Split(parent, string(os.PathSeparator))
	childParts := strings.Split(child, string(os.PathSeparator))
	if len(childParts) < len(parentParts) {
		return false
	}
	for i, part := range parentParts {
		if part != childParts[i] {
			return false
		}
	}
	return true
}

// PackageSource returns the src dir for a package
func PackageSource(gopath, pkg string) string {
	return path.Join(gopath, "src", filepath.FromSlash(pkg))
}

// PackageName returns the package name (importpath) of a path given a
// path relative to a gopath. If path is not filepath.Rel to gopath an
// error will be returned.
func PackageName(gopath, path string) (string, error) {
	path, err := filepath.Rel(gopath, path)
	if err != nil {
		return "", err
	}

	name := filepath.ToSlash(path)
	return strings.TrimPrefix(name, "src/"), nil
}

// BuildDir is the directory under the gopath that builds will be
// done in.
var BuildDir = "build"

// BuildRoot returns the root dir for building a package.
func BuildRoot(gopath, pkg string) string {
	return path.Join(gopath, BuildDir, filepath.FromSlash(pkg))
}

// PackageBuildDir returns the directory of the package in its srcdir given
// a gopath
func PackageBuildDir(gopath, pkg string) string {
	return path.Join(BuildSource(gopath, pkg), filepath.FromSlash(pkg))
}

// BuildSource returns the src dir in the build root for a package.
func BuildSource(gopath, pkg string) string {
	return path.Join(BuildRoot(gopath, filepath.FromSlash(pkg)), "src")
}

// SetupBuildRoot creates the build root for the package
// gopath/{BuildDir} and returns the root.
func SetupBuildRoot(gopath, pkg string) {
	bs := BuildRoot(gopath, pkg)
	if err := os.MkdirAll(bs, 0755); err != nil {
		log.Fatalf("Error creating directory for buildroot: %s", err.Error())
	}
}

// CopyToBuildRoot will copy a package from its home in the gopath/src
// to the build root for buildpkg.
func CopyToBuildRoot(gopath, buildPkg, pkg string) {
	bs := BuildSource(gopath, buildPkg)
	dest := path.Join(bs, pkg)
	if err := os.MkdirAll(dest, 0755); err != nil {
		log.Fatalf("Error creating directory for package in buildroot: %s", err.Error())
	}
	src := path.Join(gopath, "src", pkg)
	dc := NewDirCopier(src, dest)
	if err := dc.Copy(); err != nil {
		log.Fatalf("Error copying package %s to buildroot from %s", src, dest)
	}
}

// Verbose controls whether verbose logs will be printed from this package
var Verbose = false

// LogVerbose will log a value using log.Printf if Verbose is true.
func LogVerbose(fmtString string, args ...interface{}) {
	if Verbose {
		log.Printf(fmtString, args...)
	}
}

// MergeStringsAsSet does a union on a an b
func MergeStringsAsSet(a []string, b ...string) []string {
OuterLoop:
	for _, s := range b {
		for _, s1 := range a {
			if s == s1 {
				continue OuterLoop
			}
		}
		a = append(a, s)
	}

	return a
}
