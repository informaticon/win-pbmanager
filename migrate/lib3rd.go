package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/informaticon/lib.go.base.pborca/orca"

	_ "embed"
)

//go:embed pb_files/pbdom170.pbl
var pblPbdom170 []byte

//go:embed pb_files/pbdom115.pbl
var pblPbdom115 []byte

//go:embed pb_files/liq1.pbl
var pblLiq1 []byte

//go:embed pb_files/empty.pbl
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
			case "liq1.pbl":
				err = os.WriteFile(lib, pblLiq1, 0664)
				fmt.Printf("  add missing pbl %s\n", filepath.Base(lib))
			default:
				err = os.WriteFile(lib, pblEmpty, 0664)
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
