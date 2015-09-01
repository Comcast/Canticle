
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
}