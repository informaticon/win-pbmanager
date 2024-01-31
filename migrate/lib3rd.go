package migrate

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/informaticon/lib.go.base.pborca/orca"
)

//go:embed pb_files/*
var pbFiles embed.FS

func getPbFile(name string) []byte {
	file, err := pbFiles.ReadFile("pb_files/" + name)
	if err != nil {
		panic(err)
	}
	return file
}

type Libs3rd struct {
	copiedFiles []string
}

func (l *Libs3rd) AddMissingLibs(pbtData *orca.Pbt) error {
	for _, lib := range pbtData.LibList {
		if _, err := os.Stat(lib); os.IsNotExist(err) {
			file := filepath.Base(lib)
			switch file {
			case "pbdom170.pbl", "pbdom115.pbl":
				fmt.Printf("  add missing pbl %s\n", filepath.Base(lib))
				err = os.WriteFile(lib, getPbFile(file), 0664)
			case "exf1.pbl", "grp1.pbl", "liq1.pbl",
				"net1.pbl", "str1.pbl":
				err = os.WriteFile(lib, getPbFile(file), 0664)
				fmt.Printf("  temporarly add missing pbl %s\n", filepath.Base(lib))
				l.copiedFiles = append(l.copiedFiles, lib)
			default:
				err = os.WriteFile(lib, getPbFile("empty.pbl"), 0664)
				fmt.Printf("  temporarly add empty pbl %s to meet the requirements of the target\n", filepath.Base(lib))
				l.copiedFiles = append(l.copiedFiles, lib)
			}
			if err != nil {
				return fmt.Errorf("AddMissingLibs failed: %v", err)
			}

		}
	}

	return nil
}
func (l *Libs3rd) CleanupLibs() error {
	for len(l.copiedFiles) > 0 {
		err := os.Remove(l.copiedFiles[len(l.copiedFiles)-1])
		if err != nil {
			return fmt.Errorf("CleanupLibs failed: %v", err)
		}
		l.copiedFiles = slices.Delete(l.copiedFiles, len(l.copiedFiles)-1, len(l.copiedFiles))
	}
	return nil
}
