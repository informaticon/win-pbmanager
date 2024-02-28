package utils

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

func FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return err == nil
}

func RemoveGlob(path string) (err error) {
	contents, err := filepath.Glob(path)
	if err != nil {
		return
	}
	for _, item := range contents {
		err = os.RemoveAll(item)
		if err != nil {
			return
		}
	}
	return
}

func CopyFile(src string, dst string) error {
	fin, err := os.Open(src)
	if err != nil {
		return err
	}
	defer fin.Close()

	fout, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer fout.Close()

	_, err = io.Copy(fout, fin)
	if err != nil {
		return err
	}
	return nil
}

// GetRessource downloads a blob from an url and caches it for further use.
// It's needed to get pbdk, some pbl files and other big binary data.
func GetRessource(url string) (string, error) {
	dstFilePath := filepath.Join(os.TempDir(), "pbmigrator", path.Base(url))
	if FileExists(dstFilePath) {
		resp, err := http.Head(url)
		if err != nil {
			return "", err
		}
		// https://stackoverflow.com/questions/70603781/do-i-need-to-close-response-body-of-http-request-even-if-i-dont-read-it
		defer resp.Body.Close()

		remoteFileModTime, err := time.Parse(time.RFC1123, resp.Header.Get("Last-Modified"))
		if err != nil {
			return "", err
		}
		localFile, err := os.Stat(dstFilePath)
		if err != nil {
			return "", err
		}
		// If local is older than remote => replace it
		if localFile.ModTime().Before(remoteFileModTime) {
			err = os.Remove(dstFilePath)
			if err != nil {
				return "", err
			}
		} else {
			return dstFilePath, nil
		}
	}

	err := os.MkdirAll(filepath.Dir(dstFilePath), os.ModePerm)
	if err != nil {
		return "", err
	}

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}

	dstFile, err := os.Create(dstFilePath)
	if err != nil {
		return "", err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, resp.Body)
	return dstFilePath, err
}

// GetCommonBaseDir returns the common ancestor (dir) of two paths.
// For example with filePath1 set to /home/simon/abc and
// filePath2 set to /home/localadmin, the function returns /home.
// If filePath1 and 2 don't have a common ancestor, an empty string is returned.
func GetCommonBaseDir(filePath1, filePath2 string) string {
	components1 := strings.Split(filepath.Clean(filePath1), string(os.PathSeparator))
	components2 := strings.Split(filepath.Clean(filePath2), string(os.PathSeparator))
	var i int

	for i = 0; i < len(components1) && i < len(components2); i++ {
		if i == 0 {
			// fix C:aaa issue
			components1[0] += "/"
			components2[0] += "/"
		}
		if components1[i] != components2[i] {
			break
		}
	}
	if i == 0 {
		return ""
	}
	return filepath.Join(components1[:i]...)
}
