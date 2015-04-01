package canticle

import (
	"flag"
	"log"
	"os"
)

type Get struct {
	flags *flag.FlagSet
}

var get = &Get{
	flags: flag.NewFlagSet("get", flag.ExitOnError),
}

var GetCommand = &Command{
	Name:             "get",
	UsageLine:        "get [-insource] [-v] [-n] [package]",
	ShortDescription: "download and install dependencies as defined in Canticle file",
	LongDescription:  `The get command will fetch the dependencies for packages into each packages Canticle build root. Which is generally at $GOPATH/build/$IMPORTPATH. If -insource is specified only one package may be specified. Instead packages will be fetched into the $GOPATH as necessary and set to the correct revision. The get command may be used against both non Canticle defined (no revisions wil be set) and Canticle defined packages. If the get command is issued against a pacakge which does not exist it will also be downloaded. Specify -n to only download the target package and to not resolve target deps.`,
	Flags:            get.flags,
	Cmd:              get,
}

func (g *Get) Run(args []string) {
	insource := g.flags.Bool("insource", false, "Get the packages to the enviroment gopath rather than the build dir")
	verbose := g.flags.Bool("v", false, "Be verbose when getting stuff")
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
		GetPackage(pkg, *insource)
	}
}

// GetPackage fetches a package and all of it dependencies to either
// the buildroot or the gopath.
func GetPackage(pkg string, insource bool) {
	LogVerbose("Fetching package %s", pkg)
	gopath := os.ExpandEnv("$GOPATH")
	if !insource {
		SetupBuildRoot(gopath, pkg)
		CopyToBuildRoot(gopath, pkg, pkg)
		gopath = BuildRoot(gopath, pkg)
	}
	resolver := NewRepoDiscovery(gopath)
	depReader := &DepReader{}
	dl := NewDependencyLoader(resolver.ResolveRepo, gopath)
	dw := NewDependencyWalker(depReader.ReadDependencies, dl.FetchUpdatePackage)
	err := dw.TraversePackageDependencies(pkg)
	if err != nil {
		log.Fatalf("Error fetching packages: %s", err.Error())
	}
	LogVerbose("Package %s has remotes: %+v", pkg, dl.FetchedDeps())
}
