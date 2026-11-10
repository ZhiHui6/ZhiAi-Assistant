package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"zhi-ai-assistant/internal/buildinfo"
	"zhi-ai-assistant/internal/core/store"
	"zhi-ai-assistant/internal/i18n"
	"zhi-ai-assistant/internal/platform"
)

type AboutInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Arch        string `json:"arch"`
	DataDir     string `json:"dataDir"`
	ProjectURL  string `json:"projectUrl"`
	ReleasesURL string `json:"releasesUrl"`
	IssuesURL   string `json:"issuesUrl"`
}

type UpdateInfo struct {
	Current     string `json:"current"`
	Latest      string `json:"latest"`
	HasUpdate   bool   `json:"hasUpdate"`
	CanInstall  bool   `json:"canInstall"`
	Notes       string `json:"notes"`
	PageURL     string `json:"pageUrl"`
	DownloadURL string `json:"downloadUrl"`
	ChecksumURL string `json:"checksumUrl"`
	AssetName   string `json:"assetName"`
	Sha256      string `json:"sha256"`
	Size        int64  `json:"size"`
	PublishedAt string `json:"publishedAt"`
}

type ghRelease struct {
	TagName     string    `json:"tag_name"`
	Body        string    `json:"body"`
	HTMLURL     string    `json:"html_url"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
	Digest             string `json:"digest"`
}

func (a *App) AboutInfo() AboutInfo {
	version := normalizeVersion(buildinfo.Version)
	if version == "" {
		version = "1.1.0"
	}
	return AboutInfo{
		Name:        i18n.T(buildinfo.AppName),
		Version:     version,
		Arch:        buildinfo.Arch(),
		DataDir:     store.DataDir(),
		ProjectURL:  buildinfo.ProjectURL,
		ReleasesURL: buildinfo.ReleasesURL,
		IssuesURL:   buildinfo.IssuesURL,
	}
}

func (a *App) CheckUpdate() (UpdateInfo, error) {
	current := normalizeVersion(buildinfo.Version)
	info := UpdateInfo{Current: current}
	rel, err := fetchLatestRelease()
	if err != nil {
		return info, err
	}
	info.Latest = normalizeVersion(rel.TagName)
	if info.Latest == "" {
		return info, errors.New("发布信息缺少版本号")
	}
	info.Notes = strings.TrimSpace(rel.Body)
	info.PageURL = rel.HTMLURL
	if info.PageURL == "" {
		info.PageURL = buildinfo.ReleasesURL
	}
	if !rel.PublishedAt.IsZero() {
		info.PublishedAt = rel.PublishedAt.Local().Format("2006-01-02")
	}
	if current == "" || current == "dev" {
		info.HasUpdate = true
	} else {
		info.HasUpdate = compareVersions(info.Latest, current) > 0
	}
	exeAsset, sumAsset := pickAssets(rel.Assets)
	if exeAsset != nil {
		info.DownloadURL = exeAsset.BrowserDownloadURL
		info.AssetName = exeAsset.Name
		info.Size = exeAsset.Size
		info.Sha256 = digestSHA256(exeAsset.Digest)
	}
	if sumAsset != nil {
		info.ChecksumURL = sumAsset.BrowserDownloadURL
	}
	info.CanInstall = info.HasUpdate && info.DownloadURL != "" && (info.Sha256 != "" || info.ChecksumURL != "")
	return info, nil
}

func (a *App) InstallUpdate() error {
	if !a.settingsOnly {
		return errors.New("请在设置窗口中执行更新")
	}
	info, err := a.CheckUpdate()
	if err != nil {
		return err
	}
	if !info.HasUpdate {
		return errors.New("当前已是最新版本")
	}
	if !info.CanInstall {
		return errors.New("发布包缺少校验文件，请前往发布页手动下载")
	}
	exe, err := os.Executable()
	if err != nil {
		return errors.New("无法定位程序文件")
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return errors.New("无法定位程序文件")
	}
	tmp := exe + ".new"
	_ = os.Remove(tmp)
	if err := downloadFile(info.DownloadURL, tmp); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	want, err := resolveChecksum(info)
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}
	got, err := fileSHA256(tmp)
	if err != nil {
		_ = os.Remove(tmp)
		return errors.New("无法校验更新包")
	}
	if !strings.EqualFold(got, want) {
		_ = os.Remove(tmp)
		return errors.New("更新包校验失败，已中止")
	}

	wasRunning := platform.SidebarRunning()
	if wasRunning {
		if err := writeSettingsCommand(settingsCommand{Action: actionQuit}); err != nil {
			_ = os.Remove(tmp)
			return errors.New("无法通知侧边栏退出")
		}
		platform.NotifySettingsCommand()
		if !waitSidebarExit(8 * time.Second) {
			_ = os.Remove(tmp)
			return errors.New("侧边栏尚未退出，更新已取消")
		}
	}

	old := exe + ".old"
	_ = os.Remove(old)
	if err := os.Rename(exe, old); err != nil {
		_ = os.Remove(tmp)
		if wasRunning {
			_ = exec.Command(exe).Start()
		}
		return errors.New("无法替换程序文件，请确认程序目录可写，或前往发布页手动下载")
	}
	if err := os.Rename(tmp, exe); err != nil {
		_ = os.Rename(old, exe)
		_ = os.Remove(tmp)
		if wasRunning {
			_ = exec.Command(exe).Start()
		}
		return errors.New("写入新版本失败，已回滚")
	}
	if wasRunning {
		if err := exec.Command(exe).Start(); err != nil {
			return errors.New("更新已写入，但自动启动失败，请手动启动程序")
		}
	}
	return nil
}

func fetchLatestRelease() (ghRelease, error) {
	req, err := http.NewRequest(http.MethodGet, buildinfo.LatestReleaseAPI, nil)
	if err != nil {
		return ghRelease{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", buildinfo.AppSlug+"/"+normalizeVersion(buildinfo.Version))
	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return ghRelease{}, errors.New("无法连接更新服务，请检查网络")
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return ghRelease{}, errors.New("项目尚未发布版本")
	case http.StatusForbidden:
		return ghRelease{}, errors.New("GitHub 访问受限，请稍后重试或直接打开发布页")
	default:
		return ghRelease{}, fmt.Errorf("检查更新失败（HTTP %d）", resp.StatusCode)
	}
	var rel ghRelease
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&rel); err != nil {
		return ghRelease{}, errors.New("更新信息解析失败")
	}
	return rel, nil
}

func pickAssets(assets []ghAsset) (*ghAsset, *ghAsset) {
	var exeAsset *ghAsset
	var sumAsset *ghAsset
	for i := range assets {
		name := assets[i].Name
		lower := strings.ToLower(name)
		if strings.HasSuffix(lower, ".exe") {
			if exeAsset == nil || strings.EqualFold(name, buildinfo.ExeName) {
				exeAsset = &assets[i]
			}
		}
		if strings.HasSuffix(lower, ".sha256") {
			sumAsset = &assets[i]
		}
	}
	if exeAsset != nil {
		for i := range assets {
			if strings.EqualFold(assets[i].Name, exeAsset.Name+".sha256") {
				sumAsset = &assets[i]
			}
		}
	}
	return exeAsset, sumAsset
}

func digestSHA256(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	if strings.HasPrefix(v, "sha256:") {
		return strings.TrimPrefix(v, "sha256:")
	}
	return ""
}

func resolveChecksum(info UpdateInfo) (string, error) {
	if info.Sha256 != "" {
		return strings.ToLower(info.Sha256), nil
	}
	if info.ChecksumURL == "" {
		return "", errors.New("发布包缺少校验文件，请前往发布页手动下载")
	}
	if !strings.HasPrefix(info.ChecksumURL, "https://") {
		return "", errors.New("校验文件地址不安全，已中止")
	}
	req, err := http.NewRequest(http.MethodGet, info.ChecksumURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", buildinfo.AppSlug+"/"+normalizeVersion(buildinfo.Version))
	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.New("下载校验文件失败，请检查网络")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载校验文件失败（HTTP %d）", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return "", errors.New("读取校验文件失败")
	}
	for _, field := range strings.Fields(string(raw)) {
		if len(field) == 64 {
			if _, err := hex.DecodeString(field); err == nil {
				return strings.ToLower(field), nil
			}
		}
	}
	return "", errors.New("校验文件格式无效，请前往发布页手动下载")
}

func downloadFile(url, dst string) error {
	if !strings.HasPrefix(url, "https://") {
		return errors.New("更新地址不安全，已中止")
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return errors.New("更新地址无效")
	}
	req.Header.Set("User-Agent", buildinfo.AppSlug+"/"+normalizeVersion(buildinfo.Version))
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return errors.New("下载更新失败，请检查网络")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载更新失败（HTTP %d）", resp.StatusCode)
	}
	f, err := os.Create(dst)
	if err != nil {
		return errors.New("无法写入程序目录，请确认目录可写")
	}
	if _, err := io.Copy(f, io.LimitReader(resp.Body, 256<<20)); err != nil {
		f.Close()
		return errors.New("下载中断，请重试")
	}
	if err := f.Close(); err != nil {
		return errors.New("保存更新包失败")
	}
	return nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func waitSidebarExit(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !platform.SidebarRunning() {
			return true
		}
		time.Sleep(120 * time.Millisecond)
	}
	return !platform.SidebarRunning()
}

func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")
	return v
}

func compareVersions(a, b string) int {
	ap := versionSegments(a)
	bp := versionSegments(b)
	n := len(ap)
	if len(bp) > n {
		n = len(bp)
	}
	for i := 0; i < n; i++ {
		av := 0
		bv := 0
		if i < len(ap) {
			av = ap[i]
		}
		if i < len(bp) {
			bv = bp[i]
		}
		if av > bv {
			return 1
		}
		if av < bv {
			return -1
		}
	}
	return 0
}

func versionSegments(v string) []int {
	v = normalizeVersion(v)
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ".")
	out := make([]int, 0, len(parts))
	for _, part := range parts {
		i := 0
		for i < len(part) && part[i] >= '0' && part[i] <= '9' {
			i++
		}
		if i == 0 {
			out = append(out, 0)
			continue
		}
		n, err := strconv.Atoi(part[:i])
		if err != nil {
			n = 0
		}
		out = append(out, n)
	}
	return out
}
