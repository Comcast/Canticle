package canticle

import (
	"encoding/json"
	"io"
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

func (b *BuildInfo) WriteGoFile(w io.Writer) error {
	return BuildInfoTemplate.Execute(w, b)
}

func NewBuildInfo(gopath, pkg string) (*BuildInfo, error) {
	var bi BuildInfo

	// Decode our canticle file as a raw json message so we can embed it
	var j json.RawMessage
	f, err := os.Open(path.Join(PackageSource(gopath, pkg), "Canticle"))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	d := json.NewDecoder(f)
	if err := d.Decode(&j); err != nil {
		return nil, err
	}
	bi.CanticleDeps = &j

	// Grab our version info
	r := &LocalRepoResolver{LocalPath: gopath}
	v, err := r.ResolveRepo(pkg, nil)
	if err != nil {
		return nil, err
	}
	if bi.Revision, err = v.GetRev(); err != nil {
		return nil, err
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

var BuildInfoTemplate = template.Must(template.New("version").Parse(`
package main

import "encoding/json"

// This is GENERATED CODE from the Canticle build tool at build time
var (
	CanticleDeps = json.RawMessage(` + "`{{.DepString}}`" + `)
	BuildInfo  = struct {
		BuildTime    string
		BuildUser    string
		BuildHost    string
		Revision     string
		CanticleDeps *json.RawMessage
        }{
                "{{.BuildTime}}",
	        "{{.BuildUser}}",
                "{{.BuildHost}}",
                "{{.Revision}}",
                &CanticleDeps,
        }
)

`))
