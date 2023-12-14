package migrate

import (
	"os"
	"path/filepath"
	"slices"

	"github.com/informaticon/dev.win.base.pbmanager/utils"
)

type Libs3rd struct {
	copiedFiles []string
}

func (l *Libs3rd) CopyLibs(libFolder string) error {
	lib3rdFolder := filepath.Join(libFolder, "../lib - leere 3. Schicht")
	if !utils.FileExists(lib3rdFolder) {
		return nil
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
