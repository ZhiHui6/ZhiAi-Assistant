package store

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

type fileEntry struct {
	rel  string
	size int64
}

func snapshotTree(root string) ([]fileEntry, error) {
	var out []fileEntry
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if filepath.ToSlash(rel) == "location.json" {
			return nil
		}
		out = append(out, fileEntry{rel: rel, size: info.Size()})
		return nil
	})
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].rel < out[j].rel })
	return out, nil
}

func copyVerified(src, dst string, size int64) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	written, err := io.Copy(out, in)
	closeErr := out.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if written != size {
		return fmt.Errorf("写入不完整: %s", dst)
	}
	return nil
}

func index(entries []fileEntry) map[string]int64 {
	m := make(map[string]int64, len(entries))
	for _, e := range entries {
		m[e.rel] = e.size
	}
	return m
}

func CopyDataTree(src, dst string) error {
	if src == "" || dst == "" {
		return fmt.Errorf("路径为空")
	}
	if filepath.Clean(src) == filepath.Clean(dst) {
		return nil
	}
	srcEntries, err := snapshotTree(src)
	if err != nil {
		return err
	}
	var created []string
	rollback := func() {
		for _, p := range created {
			_ = os.Remove(p)
		}
	}
	for _, e := range srcEntries {
		target := filepath.Join(dst, e.rel)
		if existing, err := os.Stat(target); err == nil && existing.Size() == e.size {
			continue
		}
		if err := copyVerified(filepath.Join(src, e.rel), target, e.size); err != nil {
			rollback()
			return err
		}
		created = append(created, target)
	}
	dstEntries, err := snapshotTree(dst)
	if err != nil {
		rollback()
		return err
	}
	dstIndex := index(dstEntries)
	want := index(srcEntries)
	for rel, size := range want {
		got, ok := dstIndex[rel]
		if !ok {
			rollback()
			return fmt.Errorf("缺少文件: %s", rel)
		}
		if got != size {
			rollback()
			return fmt.Errorf("大小不一致: %s", rel)
		}
	}
	return nil
}

func DataTreeSize(dir string) int64 {
	entries, err := snapshotTree(dir)
	if err != nil {
		return 0
	}
	var total int64
	for _, e := range entries {
		total += e.size
	}
	return total
}
