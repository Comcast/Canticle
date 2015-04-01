package canticle

import (
	"flag"
	"log"
	"os"
	"strings"
)

type Get struct {
	flags *flag.FlagSet
}

var (
	get = &Get{
		flags: flag.NewFlagSet("get", flag.ExitOnError),
	}
	insource = get.flags.Bool("insource", false, "Get the packages to the enviroment gopath rather than the build dir")
	verbose  = get.flags.Bool("v", false, "Be verbose when getting stuff")
)
var GetCommand = &Command{
	Name:             "get",
	UsageLine:        "get [-insource] [-v] [-n] [package<,source>...]",
	ShortDescription: "download and install dependencies as defined in Canticle file",
	LongDescription:  `The get command will fetch the dependencies for packages into each packages Canticle build root. Which is generally at $GOPATH/build/$IMPORTPATH. If -insource is specified only one package may be specified. Instead packages will be fetched into the $GOPATH as necessary and set to the correct revision. The get command may be used against both non Canticle defined (no revisions wil be set) and Canticle defined packages. If the get command is issued against a pacakge which does not exist it will also be downloaded. Specify -n to only download the target package and to not resolve target deps.`,
	Flags:            get.flags,
	Cmd:              get,
}

func (g *Get) Run(args []string) {
	if err := g.flags.Parse(args); err != nil {
		return
	}
	if *verbose {
		Verbose = true
	}

	pkgs := g.flags.Args()
	if *insource && len(pkgs) > 1 {
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
		GetPackage(dep, *insource)
	}
}

// GetPackage fetches a package and all of it dependencies to either
// the buildroot or the gopath.
func GetPackage(dep *Dependency, insource bool) {
	LogVerbose("Fetching package %+v", dep)
	gopath := os.ExpandEnv("$GOPATH")
	if !insource {
		pkg := dep.ImportPath
		SetupBuildRoot(gopath, pkg)
		CopyToBuildRoot(gopath, pkg, pkg)
		gopath = BuildRoot(gopath, pkg)
	}
	resolver := NewRepoDiscovery(gopath)
	depReader := &DepReader{gopath}
	dl := NewDependencyLoader(resolver.ResolveRepo, gopath)
	dw := NewDependencyWalker(depReader.ReadDependencies, dl.FetchUpdatePackage)
	err := dw.TraverseDependencies(dep)
	if err != nil {
		log.Fatalf("Error fetching packages: %s", err.Error())
	}
	LogVerbose("Package %+v has remotes: %+v", dep, dl.FetchedDeps())
}
