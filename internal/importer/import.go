// Package importer imports several source files into several PBLs
package importer

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/informaticon/dev.win.base.pbmanager/utils"
	pborca "github.com/informaticon/lib.go.base.pborca"

	_ "embed"
)

//go:embed pbdom.pbl
var pbdomPbl []byte // can't be imported by source

// Import tries to import into multiple pbls.
// It tries to do it multiple times, so it also works from circular dependencies.
func Import(orcaServer *pborca.Orca, pbtFilePath string, srcFiles, pblFiles []string) error {
	minRun := 10
	maxRun := len(srcFiles) * 3
	lastErrCount := 5000
	var errs []error

	// To speed up the backporting process, only import pbls with error counter > 0 and keep track of errors per PBL.
	// i.e. avoid importing 5 times one PBL that is already completely imported.
	errLookUpMap := make(map[string]int)

	for iteration := range maxRun {
		t1 := time.Now()
		// collected errors of one iteration, number should decrease with each iteration
		errs = make([]error, 0)

		for i, pblFilePath := range pblFiles {
			if errCount, ok := errLookUpMap[filepath.Base(pblFilePath)]; ok {
				if errCount == 0 {
					fmt.Println("skip", filepath.Base(pblFilePath), "has already 0 import errors")
					continue
				}
			}

			errSources := processPbl(pblFilePath, srcFiles[i], pbtFilePath, orcaServer)
			fmt.Println("\terror counter:", len(errSources))
			errLookUpMap[filepath.Base(pblFilePath)] = len(errSources)
			errs = append(errs, errSources...)
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

// processPbl imports all source file of one pbl directory and returns all obtained import errors.
func processPbl(pblFilePath, srcFilePath, pbtFilePath string, orcaServer *pborca.Orca) (errs []error) {
	if filepath.Base(pblFilePath) == "pbdom.pbl" {
		fmt.Println("use embedded pbdom.pbl, skip source import")
		err := os.WriteFile(pblFilePath, pbdomPbl, 0o644)
		if err != nil {
			log.Fatal(err)
		}
		return nil
	}

	fmt.Println("backport", filepath.Base(pblFilePath))

	var foundSrcFiles []string
	err := filepath.WalkDir(srcFilePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			// first collect all files, the order of importing matters, e.g. some.bin after some.srw
			foundSrcFiles = append(foundSrcFiles, path)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	// order according to file type
	foundSrcFiles = sortSrcType(foundSrcFiles)
	sortFilesMaster(foundSrcFiles)
	fmt.Println("\tfound source files:", len(foundSrcFiles))
	fmt.Println("\tfirst source file:", filepath.Base(foundSrcFiles[0]))

	// get all .bin files, might be several; key filename, val index
	binFiles := make(map[string]int)
	for i, foundSrcFile := range foundSrcFiles {
		if filepath.Ext(foundSrcFile) == ".bin" {
			binFiles[strings.TrimSuffix(filepath.Base(foundSrcFile), filepath.Ext(foundSrcFile))] = i
		}
	}

	for _, foundSrcFile := range foundSrcFiles {
		if filepath.Ext(foundSrcFile) == ".bin" {
			continue
		}

		objName := strings.TrimSuffix(filepath.Base(foundSrcFile), filepath.Ext(filepath.Base(foundSrcFile)))
		srcData, err := utils.ReadPbSource(foundSrcFile)
		if err != nil {
			log.Fatal(err) // nothing to collect: no compile error but backport failure.
		}

		// If .bin counterpart is existent, first import only the source part up to
		// "Start of PowerBuilder Binary Data Section..." as actual object type (pbe_datawindow, pbe_window, ...)
		// In a second step call the same function immediately after containing the binary data part as PBORCA_BINARY.
		// Since bin data part is not real part of source file. ONe can simply use srcData for the first step.
		errSrc := orcaServer.SetObjSource(pbtFilePath, pblFilePath, filepath.Base(objName), srcData)
		if errSrc != nil {
			errs = append(errs, errSrc)
			// ignore trivial errors due to missing dependency resolving
			if !strings.Contains(errSrc.Error(), "Compilation failed") {
				fmt.Println("set source failed for", filepath.Base(objName), errSrc)
				fmt.Println("data:", string(srcData))
				if strings.Contains(errSrc.Error(), "connectex") { // server crashed
					panic(errSrc)
				}
			}
		}

		if _, hasBin := binFiles[strings.TrimSuffix(filepath.Base(foundSrcFile), filepath.Ext(foundSrcFile))]; hasBin {
			binFile := strings.TrimSuffix(foundSrcFile, filepath.Ext(foundSrcFile)) + ".bin"
			binSection, errGetBin := GetBinarySectionFromBin(binFile)
			if errGetBin != nil {
				errs = append(errs, fmt.Errorf("failed to set OLE binary section to matching bin file %s: %v",
					binFile, errors.Join(errGetBin, errSrc)))
			}
			errSetBin := orcaServer.SetObjBinary(pbtFilePath, pblFilePath, filepath.Base(objName), binSection)
			if errSetBin != nil {
				errs = append(errs, fmt.Errorf("failed to import binary data section in a second step %s: %v",
					binFile, errors.Join(errSetBin, errSrc)))
			}
		}
	}
	return errs
}

// sortSrcType is used to import the source file not in arbitrary order or according to their name, but according to
// their meaning to reduce import errors and make the source import faster (first e.g. are structures).
func sortSrcType(foundSourceFiles []string) []string {
	extOrder := []string{".srs", ".srq", ".srp", ".srd", ".srf", ".srm", ".sru", ".srw", ".sra", ".srj"}
	uniqueMap := make(map[string]bool)
	uniqueFiles := []string{}
	for _, fp := range foundSourceFiles {
		if !uniqueMap[fp] {
			uniqueMap[fp] = true
			uniqueFiles = append(uniqueFiles, fp)
		}
	}

	// map for extension order ranking
	rank := map[string]int{}
	for i, ext := range extOrder {
		rank[strings.ToLower(ext)] = i
	}

	sort.Slice(uniqueFiles, func(i, j int) bool {
		extI := strings.ToLower(filepath.Ext(uniqueFiles[i]))
		extJ := strings.ToLower(filepath.Ext(uniqueFiles[j]))
		rankI, okI := rank[extI]
		rankJ, okJ := rank[extJ]
		if okI && okJ {
			if rankI == rankJ {
				return uniqueFiles[i] < uniqueFiles[j]
			}
			return rankI < rankJ
		}
		if okI {
			return true
		}
		if okJ {
			return false
		}
		// if neither extension known, sort lexicographically
		return uniqueFiles[i] < uniqueFiles[j]
	})
	return uniqueFiles
}

// sortFilesMaster sorts filenames slice, prioritizing files containing "master".
// This is useful since those files are inherited more often and if imported first, it drastically reduces the number
// of compile errors when importing further source.
// TODO add more logic, like "basis" or other "naming convention" that could improve the source import.
func sortFilesMaster(files []string) {
	sort.Slice(files, func(i, j int) bool {
		baseI := filepath.Base(files[i])
		baseJ := filepath.Base(files[j])

		containsTestI := strings.Contains(baseI, "master")
		containsTestJ := strings.Contains(baseJ, "master")

		if containsTestI && !containsTestJ {
			return true
		}
		if !containsTestI && containsTestJ {
			return false
		}
		return baseI < baseJ
	})
}
