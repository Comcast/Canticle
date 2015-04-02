package canticle

import (
	"flag"
	"log"
	"os"
	"strings"
)

type Get struct {
	flags    *flag.FlagSet
	insource bool
	verbose  bool
	locals   bool
}

func NewGet() *Get {
	f := flag.NewFlagSet("get", flag.ExitOnError)
	g := &Get{flags: f}
	f.BoolVar(&g.insource, "insource", false, "Get the packages to the enviroment gopath rather than the build dir")
	f.BoolVar(&g.verbose, "v", false, "Be verbose when getting stuff")
	f.BoolVar(&g.locals, "l", false, "Prefer local copies from the $GOPATH when getting stuff")

	return g
}

var get = NewGet()

var GetCommand = &Command{
	Name:             "get",
	UsageLine:        "get [-insource] [-v] [-n] [-l] [package<,source>...]",
	ShortDescription: "download and install dependencies as defined in Canticle file",
	LongDescription: `The get command will fetch the dependencies for packages into each packages Canticle build root. Which is generally at $GOPATH/build/$IMPORTPATH. The get command may be used against both non Canticle defined (no revisions wil be set) and Canticle defined packages. If the get command is issued against a pacakge which does not exist it will also be downloaded.

If -insource is specified only one package may be specified. Instead packages will be fetched into the $GOPATH as necessary and set to the correct revision.  

Specify -n to only download the target package and to not resolve target deps. 

Specify -l to prefer local copies from $GOPATH when trying to fetch a package builds.`,
	Flags: get.flags,
	Cmd:   get,
}

func (g *Get) Run(args []string) {
	if err := g.flags.Parse(args); err != nil {
		return
	}
	if g.verbose {
		Verbose = true
	}
	defer func() { Verbose = false }()

	pkgs := g.flags.Args()
	if g.insource && len(pkgs) > 1 {
		log.Fatal("Get may not be run with -insource and multiple packages")
	}
	for _, pkg := range pkgs {
		arg := strings.Split(pkg, ",")
		imp := arg[0]
		src := ""
		if len(arg) == 2 {
			src = arg[1]
		}
		dep := &Dependency{
			ImportPath: imp,
			SourcePath: src,
		}
		GetPackage(dep, g.insource, g.locals)
	}
}

// GetPackage fetches a package and all of it dependencies to either
// the buildroot or the gopath.
func GetPackage(dep *Dependency, insource, preferLocals bool) {
	LogVerbose("Fetching package %+v", dep)
	gopath := os.ExpandEnv("$GOPATH")
	targetPath := gopath
	if !insource {
		pkg := dep.ImportPath
		SetupBuildRoot(gopath, pkg)
		CopyToBuildRoot(gopath, pkg, pkg)
		targetPath = BuildRoot(gopath, pkg)
	}

	var resolvers []RepoResolver
	lr := &LocalRepoResolver{LocalPath: gopath, RemotePath: targetPath}
	rr := &RemoteRepoResolver{targetPath}
	dr := &DefaultRepoResolver{targetPath}
	if preferLocals {
		resolvers = append(resolvers, lr, rr, dr)
	} else {
		resolvers = append(resolvers, rr, dr, lr)
	}

	resolver := NewMemoizedRepoResolver(&CompositeRepoResolver{resolvers})
	depReader := &DepReader{targetPath}
	dl := NewDependencyLoader(resolver.ResolveRepo, targetPath)
	dw := NewDependencyWalker(depReader.ReadDependencies, dl.FetchUpdatePackage)
	err := dw.TraverseDependencies(dep)
	if err != nil {
		log.Fatalf("Error fetching packages: %s", err.Error())
	}
	LogVerbose("Package %+v has remotes: %+v", dep, dl.FetchedDeps())
}
