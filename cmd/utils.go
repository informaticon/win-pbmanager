package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/informaticon/dev.win.base.pbmanager/utils"
)

func findPbtFilePath(basePath string, pbtFilePath string) (string, error) {
	if pbtFilePath == "" {
		candidates, err := filepath.Glob(fmt.Sprintf("%s/*.pbt", basePath))
		if err != nil {
			return "", err
		}
		if len(candidates) == 1 {
			pbtFilePath = candidates[0]
		} else {
			for _, candidate := range candidates {
				if filepath.Base(candidate) == "a3.pbt" {
					pbtFilePath = candidate
					break
				}
			}
		}
		if pbtFilePath == "" {
			return "", fmt.Errorf("could not find suitable PowerBuilder target in path %s", basePath)
		}
	}
	if !filepath.IsAbs(pbtFilePath) {
		pbtFilePath = filepath.Join(basePath, pbtFilePath)
	}
	if !utils.FileExists(pbtFilePath) || filepath.Ext(pbtFilePath) != ".pbt" {
		return "", fmt.Errorf("file %s does not exist or is not a pbl file", pbtFilePath)
	}
	return pbtFilePath, nil
}

func isPblFile(path string) bool {
	if !utils.FileExists(path) {
		return false
	}
	// file is xxx.pbl or xxx.pbl.r123456
	if filepath.Ext(path) == ".pbl" {
		return true
	}
	cs := strings.Split(filepath.Base(path), ".")
	if len(cs) >= 3 {
		return cs[len(cs)-2] == "pbl" && cs[len(cs)-1][:1] == "r"
	}
	return false
}
func getCleanPblFilePath(basePath, path string) (string, error) {
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		path = filepath.Join(basePath, path)
	}
	if !isPblFile(path) {
		return "", fmt.Errorf("file %s does not exist or is not a pbl file", path)
	}
	return path, nil

}
func isFile(path string) bool {
	stat, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !stat.IsDir()
}

func isDir(path string) bool {
	stat, err := os.Stat(path)
	if err != nil {
		return false
	}
	return stat.IsDir()
}
