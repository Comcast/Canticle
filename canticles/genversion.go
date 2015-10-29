package canticles

import (
	"flag"
	"log"
	"os"
)

type GenVersion struct {
	flags   *flag.FlagSet
	Verbose bool
}

func NewGenVersion() *GenVersion {
	f := flag.NewFlagSet("genversion", flag.ExitOnError)
	v := &GenVersion{
		flags: f,
	}
	f.BoolVar(&v.Verbose, "v", false, "Be verbose when getting stuff")
	return v
}

var genversion = NewGenVersion()

var GenVersionCommand = &Command{
	Name:             "genversion",
	UsageLine:        "genversion [-v] [path]",
	ShortDescription: "Generate a version go package containing revision of all current dependencies.",
	LongDescription: `The genversion command will generate a package containing all deps for path for use
in reporting version information in built applications.

Specify -v to print out a verbose set of operations instead of just errors.`,
	Flags: genversion.flags,
	Cmd:   genversion,
}

func (g *GenVersion) Run(args []string) {
	if g.Verbose {
		Verbose = true
	}
	defer func() { Verbose = false }()
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	if err := g.SaveProjectDeps(wd); err != nil {
		log.Fatal(err)
	}
}

func (g *GenVersion) SaveProjectDeps(path string) error {
	gopath, err := EnvGoPath()
	if err != nil {
		return err
	}
	s := NewSave()
	s.Resolver = &PreferLocalResolution{}
	deps, err := s.ReadDeps(gopath, path)
	if err != nil {
		return err
	}
	sources, err := s.GetSources(gopath, path, deps)
	if err != nil {
		return err
	}
	LogVerbose("Discovered sources:\n%+v", sources)
	cantdeps, err := s.Resolver.ResolveConflicts(sources)
	if err != nil {
		return err
	}
	r := &LocalRepoResolver{LocalPath: gopath}
	pkg, err := PackageName(gopath, path)
	if err != nil {
		return err
	}
	v, err := r.ResolveRepo(pkg, nil)
	if err != nil {
		return err
	}
	rev, err := v.GetRev()
	if err != nil {
		return err
	}
	LogVerbose("Resolved conflicts:\n%+v", cantdeps)
	bi, err := NewBuildInfo(rev, cantdeps)
	if err != nil {
		return err
	}
	LogVerbose("Writing version files to:%s", path)
	return bi.WriteFiles(path)
}
