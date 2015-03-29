package canticle

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path"
)

type Get struct {
	pkgs   map[string]bool
	gopath string
	deps   CanticleDependencies
}

var GetCommand = &Command{
	Name:             "get",
	UsageLine:        "get [-insource] [-n] [-x] [packages]",
	ShortDescription: "download and install dependencies as defined in Canticle file",
	LongDescription:  ``,
	Cmd:              &Get{pkgs: map[string]bool{}},
}

func (g *Get) Run(args []string) {
	g.gopath = os.ExpandEnv("$GOPATH")
	deps, err := LoadDependencies(args[0], g.gopath)
	if err != nil {
		log.Fatalf("Error loading deps file: %s", err.Error())
		return
	}
	g.deps = deps
	g.GetPackage(args[0])
	fmt.Printf("PKG %s Has Remotes: %+v\n", args[0], g.pkgs)
}

func (g *Get) GetPackage(p string) error {
	// If this package isn't on disk fetch it
	g.LoadIfNotPresent(p)

	// Load this package
	pkg, err := LoadPackage(p, g.gopath)
	if err != nil {
		fmt.Println("Error Loading Package: ", err.Error())
		return err
	}

	// Check its remotes
	remotes := pkg.RemoteImports(true)
	for _, remote := range remotes {
		g.pkgs[remote] = true
	}

	// Load its childrens remotes
	for _, remote := range remotes {
		if err := g.GetPackage(remote); err != nil {
			return err
		}
	}
	return nil
}

func (g *Get) LoadIfNotPresent(p string) error {
	s, err := os.Stat(path.Join(os.ExpandEnv("$GOPATH"), "src", p))
	switch {
	case os.IsNotExist(err):
		fmt.Println("Fetching package: ", p)
		return g.FetchRepo(p)
	case os.IsPermission(err):
		return errors.New(fmt.Sprintf("%s exists but could not be fetched", p))
	case !s.IsDir():
		return errors.New(fmt.Sprintf("%s is a file not a directory", p))
	default:
		fmt.Print(p, " already exists\n")
		return nil
	}
}

func (g *Get) FetchRepo(p string) error {
	repoRoot, err := RepoRootForImportPathWithURL(p, g.deps.SourcePath(p))
	if err != nil {
		fmt.Println("Error from reporootwithpath: ", err.Error())
		return err
	}
	rev := g.deps.Revision(repoRoot.Root)
	fmt.Printf("Reporoot: %+v\n", repoRoot)
	os.Chdir(path.Join(g.gopath, "src"))
	if rev != "" {
		err = repoRoot.VCS.CreateAtRev(repoRoot.Root, repoRoot.Repo, rev)
	} else {
		err = repoRoot.VCS.Create(repoRoot.Root, repoRoot.Repo)
	}
	if err != nil {
		fmt.Println("Error at CreateVCS: ", err.Error())
		return err
	}

	return nil
}
