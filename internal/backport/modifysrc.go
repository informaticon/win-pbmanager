package backport

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// FileRule is a combination of a Matcher for filenames, its extensions, combinations of it etc. and a Handler modifying
// those matchers.
type FileRule struct {
	Matcher func(filename string) bool
	Handler func(filename string, content []byte) ([]byte, error)
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
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		for _, rule := range rules {
			err = applyRule(rule, path, info)
		}
		return nil
	})
}

// applyRule modifies files matching the rule according to the given handler.
func applyRule(rule FileRule, path string, info os.FileInfo) error {
	if rule.Matcher(path) {
		content, err := os.ReadFile(path)
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
var regexReplaceRelease = regexp.MustCompile(`^(release.*)(\r?\n)?`)

// handleSrdFile ensures that the first line within each srd file contains "release 22;".
func handleSrdFile(filename string, content []byte) ([]byte, error) { // TODO only 25 -> 22 oar all lower also, e.g. 17?
	if bytes.Contains(content, []byte("release 25;")) {
		fmt.Printf("Modify currently set release within %s to 22\n", filename)
		content = regexReplaceRelease.ReplaceAll(content, []byte("release 22;"+"$2"))
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
