package backport

import (
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "embed"

	"github.com/informaticon/dev.win.base.pbmanager/internal/importer"
)

type Solution struct {
	Projects Projects
}

type Projects struct {
	XMLName  xml.Name  `xml:"Projects"`
	Default  string    `xml:"Default,attr"`
	Projects []Project `xml:"Project"`
}

type Project struct {
	Path     string `xml:"Path,attr"` // are relative to solution file, e.g. ./test/tst_grp.pbproj
	BasePath string `xml:"-"`         // e.g. C:\ax\lib.pb.base.graphic-utils where graphic-utils.pbsln is or .pbproj location
}

// NewSolution unmarshalls the given solution file into a structure.
func NewSolution(slnFile string) (*Solution, error) {
	data, err := os.ReadFile(slnFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read solution file %s: %v", slnFile, err)
	}
	sol := &Solution{}
	err = xml.Unmarshal(data, sol)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal solution file to structure %s: %v", slnFile, err)
	}
	for i, p := range sol.Projects.Projects {
		sol.Projects.Projects[i] = Project{
			Path:     p.Path,
			BasePath: filepath.Dir(slnFile),
		}
	}
	return sol, nil
}

func (s *Solution) ConvertToWorkspace(opts []func(*importer.MultiImport)) error {
	for _, p := range s.Projects.Projects {
		fmt.Printf("reverting project %s...\n", filepath.Base(p.Path))
		err := p.ConvertToTarget(opts)
		if err != nil {
			return fmt.Errorf("failed to convert project at %s: %v", p.Path, err)
		}
	}
	return nil
}

//go:embed template/empty.pbl
var emptyPbl []byte

func (p *Project) ConvertToTarget(opts []func(*importer.MultiImport)) error {
	pbProj, err := NewProject(filepath.Join(p.BasePath, p.Path), opts)
	if err != nil {
		return err
	}
	operationDir := filepath.Join(p.BasePath, workspaceDir, filepath.Dir(p.Path))
	err = os.MkdirAll(operationDir, 0755)
	if err != nil {
		return err
	}
	err = GetApplicationPbl(pbProj.Application.Name, pbProj.Libraries.AppEntry, operationDir)
	if err != nil {
		return fmt.Errorf("failed to obtain minimal application PBL: %v", err)
	}
	// now have myapp.pbl, further need myapp.pbt, <allLibs>.pbl empty if not present yet
	// -> this allows to just import a few and not all every time
	pbtFile := filepath.Join(operationDir, fmt.Sprintf("%s.pbt", pbProj.Application.Name))
	err = os.WriteFile(pbtFile,
		NewTarget(pbProj.Application.Name, pbProj.Libraries.AppEntry, pbProj.Libraries.GetPblPaths()).ToBytes(),
		0644)
	if err != nil {
		return fmt.Errorf("failed to write actual application target %s: %v", pbtFile, err)
	}
	// first are directories named equally to the files listed
	var srcDirs, pblFiles = []string{}, []string{}
	for _, lib := range pbProj.Libraries.GetPblPaths() {
		srcDir := filepath.Join(p.BasePath, filepath.Dir(p.Path), lib)
		pblFile := filepath.Join(operationDir, lib)
		srcDirs = append(srcDirs, srcDir)
		pblFiles = append(pblFiles, pblFile)
		// application PBL was already created separately
		if strings.Contains(lib, pbProj.Libraries.AppEntry) {
			continue
		}
		err = os.MkdirAll(filepath.Dir(pblFile), 0644)
		if err != nil {
			return err
		}
		if _, err := os.Stat(pblFile); errors.Is(err, os.ErrNotExist) {
			errWrite := os.WriteFile(pblFile, emptyPbl, 0644)
			if errWrite != nil {
				return fmt.Errorf("failed to write empty PBL %s: %v", pblFile, errWrite)
			}
		}
	}
	// have all ingredients, can start to import actual source
	err = importer.NewMultiImport(pbtFile, pblFiles, srcDirs, opts...).Import()
	if err != nil {
		return fmt.Errorf("failed to multi-import into PBLs at %s: %v", operationDir, err)
	}
	return nil
}
