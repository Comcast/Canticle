package canticle

import (
	"flag"
	"fmt"
	"log"
	"os"
)

type Get struct {
	flags        *flag.FlagSet
	Insource     bool
	Verbose      bool
	Nodeps       bool
	PreferLocals bool
}

func NewGet() *Get {
	f := flag.NewFlagSet("get", flag.ExitOnError)
	g := &Get{flags: f}
	f.BoolVar(&g.Insource, "insource", false, "Get the packages to the enviroment gopath rather than the build dir")
	f.BoolVar(&g.Verbose, "v", false, "Be verbose when getting stuff")
	f.BoolVar(&g.Nodeps, "n", false, "Only fetch the target package, do not resolve deps")
	f.BoolVar(&g.PreferLocals, "l", false, "Prefer local copies from the $GOPATH when getting stuff")

	return g
}

var get = NewGet()

var GetCommand = &Command{
	Name:             "get",
	UsageLine:        "get [-insource] [-v] [-n] [-l] [package<,source>...]",
	ShortDescription: "download dependencies as defined in Canticle file",
	LongDescription: `The get command will fetch the dependencies for packages into each packages Canticle build root. Which is generally at $GOPATH/build/$IMPORTPATH. The get command may be used against both non Canticle defined (no revisions wil be set) and Canticle defined packages. If the get command is issued against a pacakge which does not exist it will also be downloaded.

If -insource is specified only one package may be specified. Instead packages will be fetched into the $GOPATH as necessary and set to the correct revision.  

Specify -v to print out a verbose set of operations instead of just errors.

Specify -n to only download the target package and to not resolve target deps. 

Specify -l to prefer local copies from $GOPATH when trying to fetch a package.`,
	Flags: get.flags,
	Cmd:   get,
}

func (g *Get) Run(args []string) {
	if g.Verbose {
		Verbose = true
	}
	defer func() { Verbose = false }()

	pkgs := g.flags.Args()
	if g.Insource && len(pkgs) > 1 {
		log.Fatal("Get may not be run with -insource and multiple packages")
	}
	deps := ParseCmdLineDeps(pkgs)
	for _, dep := range deps {
		g.GetPackage(dep)
	}
}

// GetPackage fetches a package and all of it dependencies to either
// the buildroot or the gopath.
func (g *Get) GetPackage(dep *Dependency) error {
	LogVerbose("Fetching package %+v", dep)
	// Setup or build path
	gopath := os.ExpandEnv("$GOPATH")
	targetPath := gopath
	if !g.Insource {
		pkg := dep.ImportPath
		SetupBuildRoot(gopath, pkg)
		CopyToBuildRoot(gopath, pkg, pkg)
		targetPath = BuildRoot(gopath, pkg)
	}

	// Setup our resolvers, loaders, and walkers
	var resolvers []RepoResolver
	lr := &LocalRepoResolver{LocalPath: gopath, RemotePath: targetPath}
	rr := &RemoteRepoResolver{targetPath}
	dr := &DefaultRepoResolver{targetPath}
	if g.PreferLocals {
		resolvers = append(resolvers, lr, rr, dr)
	} else {
		resolvers = append(resolvers, rr, dr, lr)
	}
	resolver := NewMemoizedRepoResolver(&CompositeRepoResolver{resolvers})
	depReader := &DepReader{targetPath}
	dl := NewDependencyLoader(resolver.ResolveRepo, targetPath)
	dw := NewDependencyWalker(depReader.ReadDependencies, dl.FetchUpdatePackage)

	// And walk it
	err := dw.TraverseDependencies(dep)
	if err != nil {
		return fmt.Errorf("Error fetching packages: %s", err.Error())
	}
	LogVerbose("Package %+v has remotes: %+v", dep, dl.FetchedDeps())

	return nil
}
