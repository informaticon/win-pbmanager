// Package importer imports several source files into several PBLs
package importer

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/informaticon/dev.win.base.pbmanager/utils"
	pborca "github.com/informaticon/lib.go.base.pborca"

	_ "embed"
)

// Import tries to import into multiple pbls.
// It tries to do it multiple times, so it also works from circular dependencies.
func Import(orcaServer *pborca.Orca, pbtFilePath string, srcFiles, pblFiles []string) error {
	minRun := 15
	maxRun := len(srcFiles) * 3
	lastErrCount := 5000
	var errs []error
	for iteration := range maxRun {
		t1 := time.Now()
		errs = make([]error, 0)
		// collect only errors of one iteration, number should decrease with each iteration

		for i, pblFilePath := range pblFiles {
			srcFilePath := srcFiles[i]

			if filepath.Base(pblFilePath) == "pbdom.pbl" {
				fmt.Println("use embedded pbdom.pbl, skip source import")
				err := os.WriteFile(pblFilePath, pbdomPbl, 0o644)
				if err != nil {
					log.Fatal(err)
				}
				continue
			}
			fmt.Println("backport", filepath.Base(pblFilePath))

			var srcFiles []string
			err := filepath.WalkDir(srcFilePath, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if !d.IsDir() {
					// first collect all files, the order of importing matters, e.g. some.bin after some.srw
					srcFiles = append(srcFiles, path)
				}
				return nil
			})
			if err != nil {
				log.Fatal(err)
			}

			// get all .bin files, might be several; key filename, val index
			binFiles := make(map[string]int)
			for i, srcFile := range srcFiles {
				if filepath.Ext(srcFile) == ".bin" {
					binFiles[strings.TrimSuffix(filepath.Base(srcFile), filepath.Ext(srcFile))] = i
				}
			}

			for _, srcFile := range srcFiles {
				if filepath.Ext(srcFile) == ".bin" {
					continue
				}

				objName := strings.TrimSuffix(filepath.Base(srcFile), filepath.Ext(filepath.Base(srcFile)))
				srcData, err := utils.ReadPbSource(srcFile)
				if err != nil {
					errs = append(errs, err)
				}

				// If .bin counterpart is existent, first import only the source part up to
				// "Start of PowerBuilder Binary Data Section..." as actual object type (pbe_datawindow, pbe_window, ...)
				// In a second step call the same function immediately after containing the binary data part as PBORCA_BINARY.
				// Since bin data part is not real part of source file. ONe can simply use srcData for the first step.
				errSrc := orcaServer.SetObjSource(pbtFilePath, pblFilePath, filepath.Base(objName), srcData)
				if _, hasBin := binFiles[strings.TrimSuffix(filepath.Base(srcFile), filepath.Ext(srcFile))]; hasBin {
					binFile := strings.TrimSuffix(srcFile, filepath.Ext(srcFile)) + ".bin"
					binSection, err := GetBinarySectionFromBin(binFile)
					if err != nil {
						errs = append(errs, fmt.Errorf("failed to set OLE binary section to matching bin file %s: %v", binFile, errors.Join(err, errSrc)))
					}
					errBin := orcaServer.SetObjBinary(pbtFilePath, pblFilePath, filepath.Base(objName), binSection)
					if errBin != nil {
						errs = append(errs, fmt.Errorf("failed to import binary data section in a second step %s: %v", binFile, errors.Join(errBin, errSrc)))
					}
				}
				if errSrc != nil {
					errs = append(errs, errSrc)
				}
			}
		}

		fmt.Printf("Run %d took %s\n", iteration, time.Since(t1).Truncate(time.Second).String())
		if len(errs) == 0 {
			return nil
		}
		if len(errs) >= lastErrCount && iteration > minRun {
			return fmt.Errorf("compilation errors occured (multiple tries did not help): %v", errs)
		}

		lastErrCount = len(errs)
		fmt.Printf("Got %d errors in run %d. Retry...\n", lastErrCount, iteration)
	}
	return fmt.Errorf("compilation errors occured: %v", errs)
}

//go:embed pbdom.pbl
var pbdomPbl []byte // can't be imported by source; TODO: another pbdom.pbl for each project?
