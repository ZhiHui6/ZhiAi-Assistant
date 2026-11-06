package buildinfo

import "runtime"

var Version = "1.0.0"

const (
	AppName = "智爱小助手"
	AppSlug = "ZhiAi-Assistant"
	ExeName = AppSlug + ".exe"

	Repo             = "ZhiHui6/ZhiAi-Assistant"
	ProjectURL       = "https://github.com/" + Repo
	ReleasesURL      = ProjectURL + "/releases"
	IssuesURL        = ProjectURL + "/issues"
	LatestReleaseAPI = "https://api.github.com/repos/" + Repo + "/releases/latest"
)

func Arch() string {
	if runtime.GOOS == "windows" && runtime.GOARCH == "amd64" {
		return "win-x64"
	}
	return runtime.GOOS + "-" + runtime.GOARCH
}
