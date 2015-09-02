package canticles

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
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
	// Don't copy "hidden files" if we are not told to
	if !dc.CopyDot && strings.HasPrefix(filepath.Base(path), ".") {
		if f.IsDir() {
			return filepath.SkipDir
		}
		return nil
	}
	if err != nil {
		return err
	}
	// If our file isn't a directory or a normal file ignore it
	// (don't get unix domain sockets etc.)
	if !f.Mode().IsDir() && !f.Mode().IsRegular() {
		return nil
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
	d.Chmod(f.Mode())
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

// Verbose controls whether verbose logs will be printed from this package
var Verbose = false

// LogVerbose will log a value using log.Printf if Verbose is true.
func LogVerbose(fmtString string, args ...interface{}) {
	if Verbose {
		log.Printf(fmtString, args...)
	}
}

// Quite being true prevents LogWarn from printing.
var Quite = false

// LogWarn will print lines unless quite is true
func LogWarn(fmtString string, args ...interface{}) {
	if Quite {
		return
	}
	log.Printf("WARN: "+fmtString, args...)
}

// StringSets adds set like operations to a string map.
type StringSet map[string]bool

// NewStringSet returns an initalized string set.
func NewStringSet() StringSet {
	return make(map[string]bool)
}

// String so this value pretty prints well.
func (ss StringSet) String() string {
	keys := make([]string, 0, len(ss))
	for k := range ss {
		keys = append(keys, k)
	}
	return fmt.Sprintf("%+v", keys)
}

// Add strings to set.
func (ss StringSet) Add(b ...string) {
	for _, s := range b {
		ss[s] = true
	}
}

// Union performs the union of this with another string set.
func (ss StringSet) Union(sets ...StringSet) {
	for _, set := range sets {
		for str := range set {
			ss[str] = true
		}
	}
}

// Array returns the set as a sorted array.
func (ss StringSet) Array() []string {
	result := make([]string, 0, len(ss))
	for s := range ss {
		result = append(result, s)
	}
	sort.Strings(result)
	return result
}

// Size of the string set.
func (ss StringSet) Size() int {
	return len(ss)
}

// Return the location of the depedency file for a path. Should be a
// directory.
func DependencyFile(p string) string {
	return path.Join(p, "Canticle")
}

func VisibleSubDirectories(dirname string) ([]string, error) {
	finfos, err := ioutil.ReadDir(dirname)
	subdirs := make([]string, 0, len(finfos))
	for _, f := range finfos {
		if f.IsDir() && !strings.HasPrefix(f.Name(), ".") {
			subdirs = append(subdirs, path.Join(dirname, f.Name()))
		}
	}
	return subdirs, err
}

func ProjectRoot(dirname string) string {
	list := filepath.SplitList(dirname)
	for i, part := range list {
		if part == "src" {
			return path.Join(list[:i]...)
		}
	}
	return ""
}
