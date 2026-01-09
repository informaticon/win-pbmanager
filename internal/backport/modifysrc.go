package backport

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// FileRule is a combination of a Matcher for filenames, its extensions, combinations of it etc. and a Handler modifying
// those matchers.
type FileRule struct {
	description string
	Matcher     func(filename string) bool
	Handler     func(filename string, content []byte) ([]byte, error)
}

// ConvertSrcDirs modifies all files according to rules within the listed root dirs.
func ConvertSrcDirs(sourceDirs []string, rules []FileRule) error {
	for _, sourceDir := range sourceDirs {
		err := convertSrcDir(sourceDir, rules)
		if err != nil {
			return fmt.Errorf("failed to modify file within %s: %v", sourceDir, err)
		}
	}
	return nil
}

// convertSrcDir modifies all files according to rules within root.
func convertSrcDir(root string, rules []FileRule) error {
	// root might be a junction/symlink to a shorter path because pbautobuild is retarded and crashes on paths
	// with length around 180 chars.
	// Since walkDir checks for !d.IsDir() which is true on a junction the manual ReadDir is needed to access the
	// junction its content.
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		fullPath := filepath.Join(root, entry.Name())
		if entry.IsDir() {
			err = convertSrcSubDir(fullPath, rules)
			if err != nil {
				return err
			}
		} else {
			var fi fs.FileInfo
			fi, err = entry.Info()
			if err != nil {
				return err
			}
			err = convertSrcFile(fullPath, fi, rules)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// convertSrcFile applies the src conversion rules to a direct sub-root file.
func convertSrcFile(fullPath string, fi fs.FileInfo, rules []FileRule) error {
	for _, rule := range rules {
		err := applyRule(rule, fullPath, fi)
		if err != nil {
			return fmt.Errorf("rule %s could not be applied for file %s", rule.description, fullPath)
		}
	}
	return nil
}

// convertSrcSubDir applies the src conversion rules to a sub-root directory
func convertSrcSubDir(fullPath string, rules []FileRule) error {
	return filepath.WalkDir(fullPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		fi, err := d.Info()
		if err != nil {
			return err
		}

		for _, rule := range rules {
			err = applyRule(rule, path, fi)
			if err != nil {
				return fmt.Errorf("rule %s could not be applied for file %s", rule.description, path)
			}
		}
		return nil
	})
}

// applyRule modifies files matching the rule according to the given handler.
func applyRule(rule FileRule, path string, info os.FileInfo) error {
	if rule.Matcher(path) {
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		reader := transform.NewReader(file, unicode.BOMOverride(unicode.UTF8.NewDecoder()))
		content, err := io.ReadAll(reader)
		// content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		newContent, err := rule.Handler(path, content)
		if err != nil {
			return err
		}
		if !bytes.Equal(content, newContent) {
			if err = os.WriteFile(path, newContent, info.Mode()); err != nil {
				return err
			}
		}
	}
	return nil
}

// Regex to match the first line (handles both \n and \r\n)
var regexReplaceRelease = regexp.MustCompile(`(?m)^(//objectcomments.*\r?\n)?(release.*)(\r?\n)?`)

// handleSrdFile ensures that the first line within each srd file contains "release 22;".
func handleSrdFile(filename string, content []byte) ([]byte, error) { // TODO only 25 -> 22 or all lower also, e.g. 17?
	if bytes.Contains(content, []byte("release 25;")) {
		fmt.Printf("Modify currently set release within %s to 22\n", filename)
		content = regexReplaceRelease.ReplaceAll(content, []byte("${1}release 22;${3}"))
	}
	return content, nil
}

var regexReplaceRuntime = regexp.MustCompile(`(?m)^.*appruntimeversion.*\n?`)

// handleSraFile ensures that within the application.sra file(s) the right runtime version is set:
// "string appruntimeversion = "22.2.0.3356""
func handleSraFile(filename string, content []byte) ([]byte, error) {
	if !bytes.Contains(content, []byte("string appruntimeversion = \"22.2.0.3356\"")) {
		fmt.Printf("Modify currently set appruntimeversion within %s to 22.2.0.3356\n", filename)
	}
	if bytes.HasPrefix(content, []byte("//objectcomments ")) {
		content = append([]byte("$PBExportComments$"), bytes.TrimPrefix(content, []byte("//objectcomments "))...)
	}
	if !bytes.HasPrefix(content, []byte("$PBExportHeader$")) {
		content = append([]byte("$PBExportHeader$"+filepath.Base(filename)+"\r\n"), content...)
	}
	return regexReplaceRuntime.ReplaceAll(content, []byte("string appruntimeversion = \"22.2.0.3356\"\n")), nil
}

// matchExt returns a matcher func returning true if the filename matches the given extension.
func matchExt(ext string) func(string) bool {
	return func(filename string) bool {
		return strings.EqualFold(filepath.Ext(filename), ext)
	}
}
