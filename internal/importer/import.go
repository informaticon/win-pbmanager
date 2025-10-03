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

	_ "embed"

	"github.com/informaticon/dev.win.base.pbmanager/utils"
	pborca "github.com/informaticon/lib.go.base.pborca"
)

//go:embed pbdom.pbl
var pbdomPbl []byte // can't be imported by source

// Import tries to import into multiple pbls.
// It tries to do it multiple times, so it also works from circular dependencies.
func Import(orcaServer *pborca.Orca, pbtFilePath string, srcFiles, pblFiles []string) error {
	minRun := 15
	maxRun := len(srcFiles) * 3
	lastErrCount := 5000
	var errs []error

	// To speed up the backporting process, only import pbls with error counter > 0 and keep track of errors per PBL.
	// i.e. avoid importing 5 times one PBL that is already completely imported.
	errLookUpMap := make(map[string]int)
	errSrcLookUpMap := make(map[string]map[string]bool)

	lastInf1Err := -1

	for iteration := range maxRun {
		t1 := time.Now()
		// collected errors of one iteration, number should decrease with each iteration
		errs = make([]error, 0)
		for i, pblFilePath := range pblFiles {

			// skip 2 layer pbls xyz2.pbl until some iteration to bring level errors first down (TODO)
			/*filename := strings.TrimSuffix(filepath.Base(pblFilePath), filepath.Ext(pblFilePath))
			if filename[len(filename)-1:] == "2" && iteration < 5 { // 5 iteration layer one first
				fmt.Println("skip layer 2 pbl", filepath.Base(pblFilePath))
				continue
			}*/

			if errCount, ok := errLookUpMap[filepath.Base(pblFilePath)]; ok {
				if errCount == 0 {
					fmt.Println("skip", filepath.Base(pblFilePath), "has already 0 import errors")
					continue
				}
			}

			if errSrcLookUpMap[filepath.Base(pblFilePath)] == nil {
				errSrcLookUpMap[filepath.Base(pblFilePath)] = make(map[string]bool)
			}
			srcMap := errSrcLookUpMap[filepath.Base(pblFilePath)]

			// TODO first feed inf1 to reduce later errors
			if !strings.Contains(filepath.Base(pblFilePath), "inf1.pbl") && lastInf1Err == -1 {
				continue
			} else if lastInf1Err == -1 {
				errInf1 := processPbl(pblFilePath, srcFiles[i], pbtFilePath, orcaServer, &srcMap)
				fmt.Println("\terror counter:", len(errInf1))
				for len(errInf1) != lastInf1Err {
					lastInf1Err = len(errInf1)
					errInf1 = processPbl(pblFilePath, srcFiles[i], pbtFilePath, orcaServer, &srcMap)
					fmt.Println("\terror counter:", len(errInf1))
				}
			}

			errSources := processPbl(pblFilePath, srcFiles[i], pbtFilePath, orcaServer, &srcMap)
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
func processPbl(pblFilePath, srcFilePath, pbtFilePath string, orcaServer *pborca.Orca, srcMap *map[string]bool) (errs []error) {
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
		if filepath.Ext(path) == ".pblmeta" {
			return nil
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
	foundSrcFiles = sortSrcTypeName(foundSrcFiles)
	fmt.Println("\tfound source files:", len(foundSrcFiles))

	// get all .bin files, might be several; key filename, val index
	binFiles := make(map[string]int)
	for i, foundSrcFile := range foundSrcFiles {
		if filepath.Ext(foundSrcFile) == ".bin" {
			binFiles[strings.TrimSuffix(filepath.Base(foundSrcFile), filepath.Ext(foundSrcFile))] = i
		}
	}

	// TODO try all source files if it does not evolve
	preCounterSuccess := 0
	for _, v := range *srcMap {
		if v {
			preCounterSuccess++
		}
	}

	for _, foundSrcFile := range foundSrcFiles {

		// don't import source files again if they were already successful
		if isOk, ok := (*srcMap)[filepath.Base(foundSrcFile)]; ok && isOk {
			continue
		}

		if filepath.Ext(foundSrcFile) == ".bin" {
			// TODO debug
			fmt.Println("set bin file to true", filepath.Base(foundSrcFile))
			(*srcMap)[filepath.Base(foundSrcFile)] = true
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
		// TODO debug
		/*if errSrc != nil {
			fmt.Println("\tset source got error", filepath.Base(objName))
		}*/

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

		var errSetBin error
		if _, hasBin := binFiles[strings.TrimSuffix(filepath.Base(foundSrcFile), filepath.Ext(foundSrcFile))]; hasBin {
			binFile := strings.TrimSuffix(foundSrcFile, filepath.Ext(foundSrcFile)) + ".bin"
			binSection, errGetBin := GetBinarySectionFromBin(binFile)
			if errGetBin != nil {
				errs = append(errs, fmt.Errorf("failed to set OLE binary section to matching bin file %s: %v",
					binFile, errors.Join(errGetBin, errSrc)))
			}
			errSetBin = orcaServer.SetObjBinary(pbtFilePath, pblFilePath, filepath.Base(objName), binSection)
			if errSetBin != nil {
				errs = append(errs, fmt.Errorf("failed to import binary data section in a second step %s: %v",
					binFile, errors.Join(errSetBin, errSrc)))
			}
		}

		// keep track of source errors and skip the ok files in next run
		(*srcMap)[filepath.Base(foundSrcFile)] = errSrc == nil && errSetBin == nil

	}
	// TODO debug
	counterSuccess := 0
	for _, v := range *srcMap {
		if v {
			counterSuccess++
		}
	}
	fmt.Println("\tsuccess counter", counterSuccess)
	// Retry all src files again if the number of errors did not improve during this run. So for nect run ALL source
	// files are again imported. This has an effect and avoids no progress at all across the project.
	// Does this imply that a source import without error is not yet a complete import?
	if preCounterSuccess == counterSuccess && len(errs) > 0 {
		fmt.Println("\treset src files error counter: Import again all files in next run")
		for k := range *srcMap {
			(*srcMap)[k] = false
		}
	}

	return errs
}

// sortSrcTypeName is used to import the source file not in arbitrary order or according to their name, but according to
// their meaning to reduce import errors and make the source import faster (first e.g. are structures).
func sortSrcTypeName(foundSourceFiles []string) []string {
	// file type order
	extPriority := []string{".srs", ".srq", ".srp", ".srd", ".srf", ".srm", ".sru", ".srw", ".sra", ".srj"}

	extOrder := make(map[string]int)
	for i, ext := range extPriority {
		extOrder[strings.ToLower(ext)] = i
	}

	sort.Slice(foundSourceFiles, func(i, j int) bool {
		fileA := foundSourceFiles[i]
		fileB := foundSourceFiles[j]

		extA := filepath.Ext(fileA)
		extB := filepath.Ext(fileB)

		// sort by file extension priority
		orderA, inPriorityListA := extOrder[extA]
		orderB, inPriorityListB := extOrder[extB]

		if inPriorityListA && inPriorityListB {
			if orderA != orderB {
				return orderA < orderB // lower order number comes first
			}
			// If extensions have the same priority, proceed to the next rule.
		} else if inPriorityListA {
			return true // A is in the list, B is not. A comes first.
		} else if inPriorityListB {
			return false // B is in the list, A is not. B comes first.
		} else {
			// neither extension is in the priority list: Sort them alphabetically by extension.
			if extA != extB {
				return extA < extB
			}
		}

		// From here one the files have the same extension priority

		// check for "master" in the filename
		baseA := strings.TrimSuffix(filepath.Base(fileA), extA)
		baseB := strings.TrimSuffix(filepath.Base(fileB), extB)
		containsMasterA := strings.Contains(baseA, "master")
		containsMasterB := strings.Contains(baseB, "master")
		if containsMasterA != containsMasterB {
			return containsMasterA // true (contains "master") is "less than" false
		}

		// From here on the files are of the same priority and "master" status.

		/*// Sort by length of the full filename
		if len(filepath.Base(fileA)) != len(filepath.Base(fileB)) {
			return len(filepath.Base(fileA)) < len(filepath.Base(fileB)) // shorter filename comes first
		}*/ // TODO BAD idea, does increase import time drastically since unlogic import order

		// alphabetical order
		return fileA < fileB
	})
	return foundSourceFiles
}
