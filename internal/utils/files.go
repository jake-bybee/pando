package utils

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (u *Store) Unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		path := uniquePath(filepath.Join(dest, f.Name))

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, os.ModePerm)
			continue
		}

		os.MkdirAll(filepath.Dir(path), os.ModePerm)

		dstFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		srcFile, err := f.Open()
		if err != nil {
			dstFile.Close()
			return err
		}

		io.Copy(dstFile, srcFile)
		srcFile.Close()
		dstFile.Close()
	}
	return nil
}

func uniquePath(path string) string {
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)

	for i := 1; ; i++ {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return path
		}
		path = fmt.Sprintf("%s(%d)%s", base, i, ext)
	}
}

func (u *Store) TimeNow() string {
	loc, _ := time.LoadLocation(u.Timezone)
	return time.Now().In(loc).Format(time.RFC3339)
}
