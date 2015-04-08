package canticle

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Runnable is used by Command to call an item with the arguments
// pertinent to it.
type Runnable interface {
	Run(args []string)
}

// Command represents a Canticle command to be run including:
// *  Save
// *  Build
// *  Test
// *  Update
type Command struct {
	Name             string
	UsageLine        string
	ShortDescription string
	LongDescription  string
	Flags            *flag.FlagSet
	Cmd              Runnable
}

// Commands is the prebuild list of Canticle commands.
var Commands = map[string]*Command{
	"build": BuildCommand,
	"get":   GetCommand,
	"save":  SaveCommand,
}

// Usage will print the commands UsageLine and LongDescription and
// then os.Exit(2).
func (c *Command) Usage() {
	fmt.Fprintf(os.Stderr, "usage %s\n", c.UsageLine)
	fmt.Fprintf(os.Stderr, "%s\n", c.LongDescription)
	os.Exit(2)
}

// Verbose controls whether verbose logs will be printed from this package
var Verbose = false

// LogVerbose will log a value using log.Printf if Verbose is true.
func LogVerbose(fmtString string, args ...interface{}) {
	if Verbose {
		log.Printf(fmtString, args...)
	}
}

// BuildDir is the directory under the gopath that builds will be
// done in.
var BuildDir = "build"

// BuildRoot returns the root dir for building a package.
func BuildRoot(gopath, pkg string) string {
	return path.Join(gopath, "build", filepath.FromSlash(pkg))
}

// BuildSource returns the src dir in the build root for a package.
func BuildSource(gopath, pkg string) string {
	return path.Join(BuildRoot(gopath, filepath.FromSlash(pkg)), "src")
}

// PackageSource returns the src dir for a package
func PackageSource(gopath, pkg string) string {
	return path.Join(gopath, "src", filepath.FromSlash(pkg))
}

// PackageBuildDir returns the directory of the package in its srcdir given
// a gopath
func PackageBuildDir(gopath, pkg string) string {
	return path.Join(BuildSource(gopath, pkg), filepath.FromSlash(pkg))
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

// ParseCmdLineDeps parses an array of "dep,source dep,source"
// into deps for use
func ParseCmdLineDeps(args []string) []*Dependency {
	deps := make([]*Dependency, 0, len(args))
	for _, arg := range args {
		pkg := strings.Split(arg, ",")
		imp := pkg[0]
		src := ""
		if len(pkg) == 2 {
			src = pkg[1]
		}
		dep := &Dependency{
			ImportPath: imp,
			SourcePath: src,
		}
		deps = append(deps, dep)
	}
	return deps
}
