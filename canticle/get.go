package canticle

import (
	"fmt"
	"log"
	"os"
)

type Get struct {
	gopath string
	deps   Dependencies
}

var GetCommand = &Command{
	Name:             "get",
	UsageLine:        "get [-insource] [-n] [-x] [package]",
	ShortDescription: "download and install dependencies as defined in Canticle file",
	LongDescription:  ``,
	Cmd:              &Get{},
}

func (g *Get) Run(args []string) {

	resolver := NewRepoDiscovery()
	depReader := &DepReader{}
	cdl := NewDependencyLoader(resolver, depReader, os.ExpandEnv("$GOPATH"))

	deps, err := cdl.LoadAllPackageDependencies(args[0])
	if err != nil {
		log.Fatalf("Error fetching packages: %s", err.Error())
	}
	fmt.Printf("PKG %s Has Remotes: %+v\n", args[0], deps)
}
