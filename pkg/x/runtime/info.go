package runtime

import (
	"runtime"
	"runtime/debug"
	"strings"
)

type RuntimeInfo struct {
	AppName     string `json:"app.name"`
	GoVersion   string `json:"go.version"`
	GoArch      string `json:"go.arch"`
	Vcs         string `json:"vcs"`
	VcsRevision string `json:"vcs.revision"`
	VcsTime     string `json:"vcs.time"`
	Dirty       bool   `json:"dirty"`
}

var _ = ""
var BuildInfo RuntimeInfo

func init() {
	BuildInfo.Dirty = true
	BuildInfo.GoVersion = runtime.Version()
	BuildInfo.GoArch = runtime.GOARCH

	// -buildvcs=true / auto
	if info, ok := debug.ReadBuildInfo(); ok {
		paths := strings.Split(info.Path, "/")
		BuildInfo.AppName = paths[len(paths)-1]

		for _, kv := range info.Settings {
			switch kv.Key {
			case "vcs":
				BuildInfo.Vcs = kv.Value
			case "vcs.revision":
				BuildInfo.VcsRevision = kv.Value[:8]
			case "vcs.time":
				BuildInfo.VcsTime = kv.Value
			case "vcs.modified":
				BuildInfo.Dirty = kv.Value == "true"
			}
		}
	}
}
