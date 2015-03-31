package canticle

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

type DirCopier struct {
	source, dest string
	CopyDot      bool
}

func NewDirCopier(source, dest string) *DirCopier {
	return &DirCopier{source, dest, false}
}

func (dc *DirCopier) Copy() error {
	return filepath.Walk(dc.source, dc.cp)
}

func (dc *DirCopier) cp(path string, f os.FileInfo, err error) error {
	if !dc.CopyDot && strings.HasPrefix(filepath.Base(path), ".") {
		if f.IsDir() {
			return filepath.SkipDir
		}
		return nil
	}
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(dc.source, path)
	if err != nil {
		return err
	}
	if f.IsDir() {
		dest := filepath.Join(dc.dest, rel)
		return os.MkdirAll(dest, f.Mode())
	}
	s, err := os.Open(path)
	if err != nil {
		return err
	}
	defer s.Close()

	dst := filepath.Join(dc.dest, rel)
	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer d.Close()
	if _, err := io.Copy(d, s); err != nil {
		return err
	}
	return nil
}
