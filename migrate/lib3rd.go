package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/informaticon/lib.go.base.pborca/orca"

	_ "embed"
)

//go:embed pbl_files/pbdom170.pbl
var pblPbdom170 []byte

//go:embed pbl_files/pbdom115.pbl
var pblPbdom115 []byte

//go:embed pbl_files/empty.pbl
var pblEmpty []byte

type Libs3rd struct {
	copiedFiles []string
}

func (l *Libs3rd) AddMissingLibs(pbtData *orca.Pbt) error {
	for _, lib := range pbtData.LibList {
		if _, err := os.Stat(lib); os.IsNotExist(err) {
			switch filepath.Base(lib) {
			case "pbdom170.pbl":
				err = os.WriteFile(lib, pblPbdom170, 0664)
				fmt.Printf("  add missing pbl %s\n", filepath.Base(lib))
			case "pbdom115.pbl":
				err = os.WriteFile(lib, pblPbdom115, 0664)
				fmt.Printf("  add missing pbl %s\n", filepath.Base(lib))
			default:
				err = os.WriteFile(lib, pblEmpty, 0664)
				fmt.Printf("  temporarly add empty pbl %s to meet the requirements of the target\n", filepath.Base(lib))
				l.copiedFiles = append(l.copiedFiles, lib)
			}
			if err != nil {
				return err
			}

		}
	}
	return nil
}
func (l *Libs3rd) CleanupLibs() error {
	for len(l.copiedFiles) > 0 {
		err := os.Remove(l.copiedFiles[len(l.copiedFiles)-1])
		if err != nil {
			return err
		}
		l.copiedFiles = slices.Delete(l.copiedFiles, len(l.copiedFiles)-1, len(l.copiedFiles))
	}
	return nil
}

/*
// Copy3rdLayerLibs adds missing 3rd layer libs to the project folder.
func (l *Libs3rd) Copy3rdLayerLibs(libFolder string) error {
	var lib3rdFolder string
	if filepath.Base(libFolder) == "lib" {
		lib3rdFolder = filepath.Join(libFolder, "../lib - leere 3. Schicht")
		if !utils.FileExists(lib3rdFolder) {
			lib3rdFolder = filepath.Join(libFolder, "../lib - leere 2. und 3. Schicht")
			if !utils.FileExists(lib3rdFolder) {
				return nil
			}
		}
	} else if filepath.Base(libFolder) == "lib_lohn" {
		lib3rdFolder = filepath.Join(libFolder, "../lib_lohn - leere 3. Schicht")
		if !utils.FileExists(lib3rdFolder) {
			return nil
		}
	}

	files, err := filepath.Glob(filepath.Join(lib3rdFolder) + "/*.pbl")
	if err != nil {
		return err
	}
	for _, srcFile := range files {
		dstFile := filepath.Join(libFolder, filepath.Base(srcFile))
		if utils.FileExists(dstFile) {
			continue
		}
		err := utils.CopyFile(srcFile, dstFile)
		if err != nil {
			return err
		}
		l.copiedFiles = append(l.copiedFiles, dstFile)
	}
	return nil
}

*/
