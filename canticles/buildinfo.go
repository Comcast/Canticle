package canticles

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"text/template"
	"time"
)

type BuildInfo struct {
	BuildTime    string
	BuildUser    string
	BuildHost    string
	Revision     string
	CanticleDeps *json.RawMessage
}

func (b *BuildInfo) DepString() string {
	s, _ := json.Marshal(b.CanticleDeps)
	return string(s)
}

func (b *BuildInfo) WriteFiles(dir string) error {
	pkgdir := path.Join(dir, "buildinfo")
	if err := os.MkdirAll(pkgdir, 0755); err != nil {
		return err
	}
	if err := ioutil.WriteFile(path.Join(pkgdir, "buildinfo.go"), []byte(BuildInfoGoFile), 0644); err != nil {
		return err
	}
	f, err := os.Create(path.Join(pkgdir, "info.go"))
	if err != nil {
		return err
	}
	defer f.Close()
	return BuildInfoTemplate.Execute(f, b)
}

func NewBuildInfo(rev string, stable bool, deps []*CanticleDependency) (*BuildInfo, error) {
	var bi BuildInfo
	bi.Revision = rev

	// Encode our deps to place be placed
	msg, err := json.MarshalIndent(deps, "", "    ")
	if err != nil {
		return nil, err
	}

	// Decode our canticle file as a raw json message so we can embed it
	var j json.RawMessage
	if err := json.Unmarshal(msg, &j); err != nil {
		return nil, err
	}
	bi.CanticleDeps = &j

	// Add the other fields if a stable build info set is requiered
	if stable {
		return &bi, nil
	}

	u, err := user.Current()
	if err != nil {
		return nil, err
	}
	bi.BuildUser = u.Username

	if bi.BuildHost, err = os.Hostname(); err != nil {
		return nil, err
	}

	bi.BuildTime = time.Now().Format(time.RFC3339)

	return &bi, nil
}

var BuildInfoGoFile = `
// Package buildinfo is GENERATED CODE from the Canticle build tool,
// you may check this file in so build can happen without
// genversion. DO NOT CHECK IN info.go in this package.
package buildinfo

import "encoding/json"

// BuildInfo contains the deps of this as well as information about
// when genversion was called.
type BuildInfo struct {
	BuildTime    string
	BuildUser    string
	BuildHost    string
        Revision     string
	CanticleDeps *json.RawMessage
}

var buildInfo = &BuildInfo{}

// GetBuildInfo returns the information saved by cant genversion.
func GetBuildInfo() *BuildInfo {
        return buildInfo
}`

var BuildInfoTemplate = template.Must(template.New("version").Parse(`
package buildinfo

import "encoding/json"

// This is GENERATED CODE, DO NOT CHECK THIS IN
func init() {
	CanticleDeps := json.RawMessage(` + "`{{.DepString}}`" + `)
	buildInfo = &BuildInfo{
                "{{.BuildTime}}",
	        "{{.BuildUser}}",
                "{{.BuildHost}}",
                "{{.Revision}}",
                &CanticleDeps,
        }
}`))
