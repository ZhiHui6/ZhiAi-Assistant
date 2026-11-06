package store

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var backupSkipNames = map[string]bool{
	"usage-debug.log":   true,
	"settings.cmd.json": true,
	"settings.nav.json": true,
}

const backupManifestName = "manifest.json"

type BackupCategory struct {
	ID    string
	Files []string
	Dirs  []string
}

var BackupCategories = []BackupCategory{
	{ID: "todos", Files: []string{"todos.json"}},
	{ID: "notes", Files: []string{"notes.json"}},
	{ID: "shelf", Files: []string{"shelf.json", "mounts.json"}},
	{ID: "launch", Files: []string{"launch.json"}},
	{ID: "timers", Files: []string{"timers.json"}},
	{ID: "usage", Files: []string{"usage.json"}},
	{ID: "usageAuth", Files: []string{"usage.credentials", "workbuddy.at-rest.key"}},
	{ID: "settings", Files: []string{"settings.json"}},
	{ID: "holidays", Dirs: []string{"holidays"}},
}

type backupManifest struct {
	Schema     int      `json:"schema"`
	App        string   `json:"app"`
	CreatedAt  string   `json:"createdAt"`
	Categories []string `json:"categories"`
}

func categoryOf(rel string) string {
	clean := filepath.ToSlash(rel)
	for _, c := range BackupCategories {
		for _, name := range c.Files {
			if clean == name {
				return c.ID
			}
		}
		for _, dir := range c.Dirs {
			if strings.HasPrefix(clean, dir+"/") {
				return c.ID
			}
		}
	}
	return ""
}

func selectedCategorySet(selected []string) map[string]bool {
	set := make(map[string]bool, len(selected))
	for _, id := range selected {
		for _, c := range BackupCategories {
			if c.ID == id {
				set[id] = true
				break
			}
		}
	}
	return set
}

func isManifestName(name string) bool {
	return filepath.ToSlash(name) == backupManifestName
}

func categoryList(present map[string]bool) []string {
	out := make([]string, 0, len(present))
	for _, c := range BackupCategories {
		if present[c.ID] {
			out = append(out, c.ID)
		}
	}
	return out
}

func DataRoot() string { return filepath.Join(DataDir(), "data") }

func BackupRoot() string { return filepath.Join(DataDir(), "backup") }

func skipBackupFile(rel string) bool {
	base := filepath.Base(rel)
	if backupSkipNames[base] {
		return true
	}
	lower := strings.ToLower(base)
	return strings.HasSuffix(lower, ".tmp") || strings.HasSuffix(lower, ".zip")
}

func backupEntries(src string, chosen map[string]bool, selective bool) ([]fileEntry, map[string]bool, error) {
	all, err := snapshotTree(src)
	if err != nil {
		return nil, nil, err
	}
	out := make([]fileEntry, 0, len(all))
	present := make(map[string]bool)
	for _, e := range all {
		if skipBackupFile(e.rel) || isManifestName(e.rel) {
			continue
		}
		category := categoryOf(e.rel)
		if selective && (category == "" || !chosen[category]) {
			continue
		}
		if category != "" {
			present[category] = true
		}
		out = append(out, e)
	}
	return out, present, nil
}

type BackupInfo struct {
	Path  string
	Bytes int64
	Files int
}

func backupName(prefix string, ext string) string {
	stamp := time.Now().Format("20060102-150405")
	name := fmt.Sprintf("%s-%s%s", prefix, stamp, ext)
	for i := 1; ; i++ {
		if _, err := os.Stat(name); errors.Is(err, os.ErrNotExist) {
			return name
		}
		name = fmt.Sprintf("%s-%s-%d%s", prefix, stamp, i, ext)
	}
}

func CreateBackup(dstDir string, selected []string) (BackupInfo, error) {
	if err := ValidateDir(dstDir); err != nil {
		return BackupInfo{}, err
	}
	chosen := selectedCategorySet(selected)
	selective := selected != nil
	if selective && len(chosen) == 0 {
		return BackupInfo{}, errors.New("未选择要备份的内容")
	}
	src := DataRoot()
	entries, present, err := backupEntries(src, chosen, selective)
	if err != nil {
		return BackupInfo{}, err
	}
	if len(entries) == 0 {
		return BackupInfo{}, errors.New("所选内容没有可备份的数据")
	}
	final := filepath.Join(dstDir, backupName("zhi-ai-backup", ".zip"))
	tmp := final + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return BackupInfo{}, err
	}
	zw := zip.NewWriter(out)
	manifest := backupManifest{
		Schema:     1,
		App:        "ZhiAi-Assistant",
		CreatedAt:  time.Now().Format(time.RFC3339),
		Categories: categoryList(present),
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		_ = zw.Close()
		_ = out.Close()
		_ = os.Remove(tmp)
		return BackupInfo{}, err
	}
	mw, err := zw.Create(backupManifestName)
	if err != nil {
		_ = zw.Close()
		_ = out.Close()
		_ = os.Remove(tmp)
		return BackupInfo{}, err
	}
	if _, err := mw.Write(raw); err != nil {
		_ = zw.Close()
		_ = out.Close()
		_ = os.Remove(tmp)
		return BackupInfo{}, err
	}
	written := 0
	var total int64
	for _, e := range entries {
		f, err := os.Open(filepath.Join(src, e.rel))
		if err != nil {
			_ = zw.Close()
			_ = out.Close()
			_ = os.Remove(tmp)
			return BackupInfo{}, err
		}
		w, err := zw.Create(filepath.ToSlash(e.rel))
		if err != nil {
			_ = f.Close()
			_ = zw.Close()
			_ = out.Close()
			_ = os.Remove(tmp)
			return BackupInfo{}, err
		}
		n, err := io.Copy(w, f)
		_ = f.Close()
		if err != nil {
			_ = zw.Close()
			_ = out.Close()
			_ = os.Remove(tmp)
			return BackupInfo{}, err
		}
		written++
		total += n
	}
	if err := zw.Close(); err != nil {
		_ = out.Close()
		_ = os.Remove(tmp)
		return BackupInfo{}, err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return BackupInfo{}, err
	}
	if err := os.Rename(tmp, final); err != nil {
		_ = os.Remove(tmp)
		return BackupInfo{}, err
	}
	return BackupInfo{Path: final, Bytes: total, Files: written}, nil
}

func CreateAutoBackup() (BackupInfo, error) {
	dir := BackupRoot()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return BackupInfo{}, err
	}
	return CreateBackup(dir, nil)
}

func safeJoin(root, name string) (string, error) {
	if name == "" || strings.HasPrefix(name, "/") || strings.HasPrefix(name, "\\") ||
		filepath.IsAbs(name) || filepath.VolumeName(name) != "" {
		return "", fmt.Errorf("备份内路径非法: %s", name)
	}
	normalized := filepath.FromSlash(name)
	for _, seg := range strings.Split(normalized, string(filepath.Separator)) {
		if seg == ".." {
			return "", fmt.Errorf("备份内路径越界: %s", name)
		}
	}
	target := filepath.Join(root, normalized)
	rel, err := filepath.Rel(root, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("备份内路径越界: %s", name)
	}
	return target, nil
}

func VerifyBackup(path string) (int, int64, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return 0, 0, errors.New("无法读取备份文件")
	}
	defer zr.Close()
	count := 0
	var total int64
	for _, f := range zr.File {
		if f.FileInfo().IsDir() || isManifestName(f.Name) {
			continue
		}
		if _, err := safeJoin(string(filepath.Separator), f.Name); err != nil {
			return 0, 0, err
		}
		count++
		total += int64(f.UncompressedSize64)
	}
	if count == 0 {
		return 0, 0, errors.New("备份文件为空")
	}
	return count, total, nil
}

type BackupContents struct {
	Categories []BackupCategoryInfo
	Legacy     bool
}

type BackupCategoryInfo struct {
	ID    string
	Files int
	Bytes int64
}

func ReadBackupContents(path string) (BackupContents, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return BackupContents{}, errors.New("无法读取备份文件")
	}
	defer zr.Close()
	counts := make(map[string]*BackupCategoryInfo)
	hasManifest := false
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if isManifestName(f.Name) {
			hasManifest = true
			continue
		}
		category := categoryOf(f.Name)
		if category == "" {
			continue
		}
		info := counts[category]
		if info == nil {
			info = &BackupCategoryInfo{ID: category}
			counts[category] = info
		}
		info.Files++
		info.Bytes += int64(f.UncompressedSize64)
	}
	out := BackupContents{Legacy: !hasManifest}
	for _, c := range BackupCategories {
		if info := counts[c.ID]; info != nil && info.Files > 0 {
			out.Categories = append(out.Categories, *info)
		}
	}
	if len(out.Categories) == 0 {
		return BackupContents{}, errors.New("备份中没有可导入的数据")
	}
	return out, nil
}

func RestoreBackup(zipPath string, selected []string) (BackupInfo, error) {
	if _, err := os.Stat(zipPath); err != nil {
		return BackupInfo{}, errors.New("备份文件不存在")
	}
	wantCount, _, err := VerifyBackup(zipPath)
	if err != nil {
		return BackupInfo{}, err
	}
	pre, err := CreateAutoBackup()
	if err != nil {
		return BackupInfo{}, err
	}
	if selected != nil {
		return pre, restoreSelected(zipPath, selected)
	}
	base := DataDir()
	stamp := time.Now().Format("20060102-150405")
	tmpDir := filepath.Join(base, "data.new-"+stamp)
	oldDir := filepath.Join(base, "data.old-"+stamp)
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return pre, err
	}
	extracted, copyErr := extractZip(zipPath, tmpDir)
	if copyErr == nil && extracted != wantCount {
		copyErr = fmt.Errorf("解压文件数不符: %d / %d", extracted, wantCount)
	}
	if copyErr != nil {
		_ = os.RemoveAll(tmpDir)
		return pre, copyErr
	}
	src := DataRoot()
	if err := os.Rename(src, oldDir); err != nil {
		_ = os.RemoveAll(tmpDir)
		return pre, errors.New("无法切换数据目录，已保持原样")
	}
	if err := os.Rename(tmpDir, src); err != nil {
		_ = os.Rename(oldDir, src)
		_ = os.RemoveAll(tmpDir)
		return pre, errors.New("无法切换数据目录，已恢复原数据")
	}
	_ = os.RemoveAll(oldDir)
	return pre, nil
}

func restoreSelected(zipPath string, selected []string) error {
	chosen := selectedCategorySet(selected)
	if len(chosen) == 0 {
		return errors.New("未选择要导入的内容")
	}
	tmpDir, err := os.MkdirTemp(DataDir(), "data.pick-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	if _, err := extractZip(zipPath, tmpDir); err != nil {
		return err
	}
	dstRoot := DataRoot()
	copied := 0
	for _, c := range BackupCategories {
		if !chosen[c.ID] {
			continue
		}
		for _, name := range c.Files {
			src := filepath.Join(tmpDir, filepath.FromSlash(name))
			if _, err := os.Stat(src); err != nil {
				continue
			}
			if err := copyFileAtomic(src, filepath.Join(dstRoot, filepath.FromSlash(name))); err != nil {
				return err
			}
			copied++
		}
		for _, dir := range c.Dirs {
			n, err := copyDirInto(filepath.Join(tmpDir, filepath.FromSlash(dir)), filepath.Join(dstRoot, filepath.FromSlash(dir)))
			if err != nil {
				return err
			}
			copied += n
		}
	}
	if copied == 0 {
		return errors.New("备份中没有所选内容")
	}
	return nil
}

func copyFileAtomic(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	tmp := dst + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dst)
}

func copyDirInto(srcDir, dstDir string) (int, error) {
	count := 0
	err := filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		if err := copyFileAtomic(path, filepath.Join(dstDir, rel)); err != nil {
			return err
		}
		count++
		return nil
	})
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	return count, err
}

func extractZip(zipPath, dst string) (int, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return 0, errors.New("无法读取备份文件")
	}
	defer zr.Close()
	count := 0
	for _, f := range zr.File {
		if f.FileInfo().IsDir() || isManifestName(f.Name) {
			continue
		}
		target, err := safeJoin(dst, f.Name)
		if err != nil {
			return count, err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return count, err
		}
		rc, err := f.Open()
		if err != nil {
			return count, err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			_ = rc.Close()
			return count, err
		}
		written, err := io.Copy(out, rc)
		closeErr := out.Close()
		_ = rc.Close()
		if err != nil {
			return count, err
		}
		if closeErr != nil {
			return count, closeErr
		}
		if uint64(written) != f.UncompressedSize64 {
			return count, fmt.Errorf("解压不完整: %s", f.Name)
		}
		count++
	}
	return count, nil
}

func ListBackups(dir string, limit int) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".zip") {
			continue
		}
		out = append(out, filepath.Join(dir, e.Name()))
	}
	sortDescByTime(out)
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func sortDescByTime(paths []string) {
	mod := func(p string) time.Time {
		info, err := os.Stat(p)
		if err != nil {
			return time.Time{}
		}
		return info.ModTime()
	}
	for i := 1; i < len(paths); i++ {
		for j := i; j > 0 && mod(paths[j]).After(mod(paths[j-1])); j-- {
			paths[j], paths[j-1] = paths[j-1], paths[j]
		}
	}
}
