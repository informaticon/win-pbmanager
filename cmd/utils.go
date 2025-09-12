package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/informaticon/dev.win.base.pbmanager/utils"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
)

func findPbtFilePath(basePath string, pbtFilePath string) (string, error) {
	return findFilePath(basePath, ".pbt", pbtFilePath)
}

func findPbProjFilePath(basePath string, pbtFilePath string) (string, error) {
	return findFilePath(basePath, "pbproj", pbtFilePath)
}

// findFilePath searches for files with a given file extension in basePath.
// extension must be prefixed with a dot (e.g. '.pbt').
// If initPath is not an empty string, the function checks if it matches the criterias
//
//	(has the right extension and does exist) and returns it without looking of other files.
func findFilePath(basePath string, extension string, initPath string) (string, error) {
	var err error
	if initPath != "" {
		if !filepath.IsAbs(initPath) {
			initPath, err = filepath.Abs(filepath.Join(basePath, initPath))
			if err != nil {
				return "", err
			}
		}
		if !utils.FileExists(initPath) || filepath.Ext(initPath) != extension {
			return "", fmt.Errorf("file %s does not exist or is not a %s file", initPath, extension)
		}
	}
	candidates, err := filepath.Glob(basePath + "/*" + extension)
	if err != nil {
		return "", err
	}
	if len(candidates) == 1 {
		initPath = candidates[0]
	} else {
		for _, candidate := range candidates {
			if filepath.Base(candidate) == "a3.pbt" {
				initPath = candidate
				break
			}
		}
	}
	if initPath == "" {
		return "", fmt.Errorf("could not find suitable PowerBuilder target in path %s", basePath)
	}
	return initPath, nil
}

func isPblPbtFile(path string) bool {
	if !utils.FileExists(path) {
		return false
	}
	// file is xxx.pbl or xxx.pbl.r123456
	if filepath.Ext(path) == ".pbl" || filepath.Ext(path) == ".pbt" {
		return true
	}
	cs := strings.Split(filepath.Base(path), ".")
	if len(cs) >= 3 {
		return (cs[len(cs)-2] == "pbl" || cs[len(cs)-2] == "pbt") && cs[len(cs)-1][:1] == "r"
	}
	return false
}

func getCleanPblPbtFilePath(basePath, path string) (string, error) {
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		path = filepath.Join(basePath, path)
	}
	path = strings.ToLower(path)
	if !isPblPbtFile(path) {
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

var encodings = map[string]encoding.Encoding{
	"utf8":    unicode.UTF8,
	"utf8bom": unicode.UTF8BOM,
	"utf16":   unicode.UTF16(unicode.LittleEndian, unicode.ExpectBOM),
	"utf16le": unicode.UTF16(unicode.LittleEndian, unicode.UseBOM),
	"cp1252":  charmap.Windows1252,
}

func encode(str string, enc string) ([]byte, error) {
	if encoder, ok := encodings[enc]; ok {
		return encoder.NewEncoder().Bytes([]byte(str))
	}
	return nil, fmt.Errorf("unknown encoding %s", enc)
}
