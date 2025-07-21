// Package backport expects a solution file and for each listed project a target and related libraries are created
// so that the target can be compiled with PB2022.
package backport

import (
	"os"
	"path/filepath"
	"regexp"

	"github.com/informaticon/dev.win.base.pbmanager/internal/importer"
	"github.com/informaticon/dev.win.base.pbmanager/utils"
)

const workspaceDir = "workspace"

// ConvertProjectToTarget modifies src files referenced by .pbproj directory and converts the project back to target.
func ConvertProjectToTarget(pbProjFile string, opts []func(*importer.MultiImport)) error {
	rules := []FileRule{
		{Matcher: matchExt(".srd"), Handler: handleSrdFile},
		{Matcher: matchExt(".sra"), Handler: handleSraFile},
	}
	err := ConvertSrcDirs([]string{filepath.Dir(pbProjFile)}, rules)
	if err != nil {
		return err
	}
	p := &Project{
		Path:     filepath.Base(pbProjFile),
		BasePath: filepath.Dir(pbProjFile),
	}
	return p.ConvertToTarget(opts)
}

// ConvertSolutionToWorkspace modifies src files within .pbsln directory and converts the solution back to workspace.
func ConvertSolutionToWorkspace(slnFile string, opts []func(*importer.MultiImport)) error {
	rules := []FileRule{
		{Matcher: matchExt(".srd"), Handler: handleSrdFile},
		{Matcher: matchExt(".sra"), Handler: handleSraFile},
	}
	err := ConvertSrcDirs([]string{filepath.Dir(slnFile)}, rules)
	s, err := NewSolution(slnFile)
	if err != nil {
		return err
	}
	err = s.ConvertToWorkspace(opts)
	if err != nil {
		return err
	}
	return copyAssetsToWorkspace(slnFile)
}

// copyAssetsToWorkspace copies assets from solution to workspace so that it can be fully used, e.g. dlls, test files...
func copyAssetsToWorkspace(slnFile string) error {
	subDirs, err := utils.ImmediateSubDirs(filepath.Dir(slnFile))
	if err != nil {
		return err
	}
	for _, subDir := range subDirs {
		err = copyDir(filepath.Join(filepath.Dir(slnFile), subDir),
			filepath.Join(filepath.Dir(slnFile), workspaceDir, subDir), defaultIgnoreFunc)
		if err != nil {
			return err
		}
	}
	return nil
}

// IgnoreFunc defines the signature for custom ignore logic.
// Return true to ignore (skip) the path, false to include it.
type IgnoreFunc func(d os.DirEntry) bool

// copyDir copies the contents of srcDir to dstDir, using the provided ignore function.
func copyDir(srcDir, dstDir string, ignore IgnoreFunc) error {
	return filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dstDir, relPath)

		if ignore != nil && ignore(d) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}
		return utils.CopyFile(path, targetPath)
	})
}

var regDirIgnore = regexp.MustCompile(`.*\.pbl|.pb|build|_BackupFiles|workspace`)
var regFileIgnore = regexp.MustCompile(`.pbproj|.pbl|.pbsln|.opt`)

// ignoreFunc copies back all defined exceptions for directories and
func defaultIgnoreFunc(d os.DirEntry) bool {
	if d.IsDir() && regDirIgnore.MatchString(d.Name()) {
		return true
	}
	if !d.IsDir() && regFileIgnore.MatchString(d.Name()) {
		return true
	}
	return false
}
