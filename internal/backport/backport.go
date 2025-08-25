// Package backport expects a solution file and for each listed project a target and related libraries are created
// so that the target can be compiled with PB2022.
package backport

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/informaticon/dev.win.base.pbmanager/internal/importer"
	pborca "github.com/informaticon/lib.go.base.pborca"
	"github.com/informaticon/lib.go.base.pborca/pbtemplates"
)

const workspaceDir = "workspace"

// ConvertProjectToTarget modifies src files referenced by .pbproj directory and converts the project back to target.
func ConvertProjectToTarget(Orca *pborca.Orca, pbProjFile string) error {
	rules := []FileRule{
		{Matcher: matchExt(".srd"), Handler: handleSrdFile},
		{Matcher: matchExt(".sra"), Handler: handleSraFile},
	}
	err := ConvertSrcDirs([]string{filepath.Dir(pbProjFile)}, rules)
	if err != nil {
		return err
	}
	pbProj, err := NewProject(pbProjFile)
	if err != nil {
		return err
	}
	workDir := filepath.Join(filepath.Dir(pbProjFile), workspaceDir)
	err = os.MkdirAll(workDir, 0o755)
	if err != nil {
		return err
	}

	// Create pbt file
	pbtFilePath := filepath.Join(workDir, pbProj.Application.Name+".pbt")
	err = os.WriteFile(pbtFilePath,
		NewTarget(pbProj.Application.Name, pbProj.Libraries.AppEntry, pbProj.Libraries.GetPblPaths()).ToBytes(),
		0o644)
	if err != nil {
		return fmt.Errorf("failed to write actual application target %s: %v", pbtFilePath, err)
	}
	// Create main pbl file (needed, because Orca only works if application is already compilable)
	err = createMainPblFromPbProj(Orca, pbProj, workDir)
	if err != nil {
		return err
	}

	// first are directories named equally to the files listed
	srcDirs, pblFiles := []string{}, []string{}
	for i, lib := range pbProj.Libraries.GetPblPaths() {
		srcDirs = append(srcDirs, filepath.Join(filepath.Dir(pbProjFile), lib))
		pblFiles = append(pblFiles, filepath.Join(workDir, lib))
		// application PBL was already created separately
		if strings.Contains(lib, pbProj.Libraries.AppEntry) {
			continue
		}
		err = os.MkdirAll(filepath.Dir(pblFiles[i]), 0o644)
		if err != nil {
			return err
		}
		if _, err := os.Stat(pblFiles[i]); errors.Is(err, os.ErrNotExist) {
			errWrite := os.WriteFile(pblFiles[i], pbtemplates.GetEmptyPbl(), 0o644)
			if errWrite != nil {
				return fmt.Errorf("failed to write empty PBL %s: %v", pblFiles[i], errWrite)
			}
		}
	}

	// have all ingredients, can start to import actual source
	err = importer.Import(Orca, pbtFilePath, srcDirs, pblFiles)
	if err != nil {
		return fmt.Errorf("failed to multi-import into PBLs at %s: %v", workDir, err)
	}
	return nil
}

func createMainPblFromPbProj(Orca *pborca.Orca, pbProj *PbProject, workDir string) error {
	pblSrc, err := Orca.CreateApplicationPbl(pbProj.Application.Name, pbtemplates.GenerateSra(pbProj.Application.Name))
	if err != nil {
		return fmt.Errorf("failed to obtain minimal application PBL: %v", err)
	}
	err = os.WriteFile(filepath.Join(workDir, pbProj.Libraries.AppEntry), pblSrc, 0o644)
	if err != nil {
		return fmt.Errorf("failed to obtain minimal application PBL: %v", err)
	}
	return nil
}
