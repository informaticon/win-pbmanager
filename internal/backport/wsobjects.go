package backport

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/informaticon/dev.win.base.pbmanager/internal/importer"
	"github.com/informaticon/dev.win.base.pbmanager/utils"
)

// Src25ToWsObjects moves all xyz.pbl directories beside the .pbproj file to ws_objects/xyz.pbl.src
// and integrates all .bin files into .sr* files so that pbautobuild220 can regenerate the PBLs.
func Src25ToWsObjects(pbProj *PbProject) error {
	// all layers must be created als src dir but empty, e.g. adi3.pbl.src
	wsObjects := filepath.Join(filepath.Dir(pbProj.filePath), "ws_objects")
	err := os.MkdirAll(wsObjects, 0o755)
	if err != nil {
		return err
	}
	for _, lib := range pbProj.Libraries.GetPblPaths() {
		pblDir := filepath.Join(filepath.Dir(pbProj.filePath), lib)
		srcDirWsObjects := filepath.Join(wsObjects, filepath.Base(lib+".src"))
		err = os.MkdirAll(srcDirWsObjects, 0o755)
		if err != nil {
			return err
		}
		err = utils.CopyDirectory(pblDir, srcDirWsObjects)
		if err != nil {
			return fmt.Errorf("failed to copy %s to %s: %v", pblDir, srcDirWsObjects, err)
		}
		err = os.Rename(pblDir, pblDir+".old")
		if err != nil {
			return err
		}
	}

	wsObjectsSubDirs, err := utils.ImmediateSubDirs(wsObjects)
	if err != nil {
		return err
	}
	for _, subDir := range wsObjectsSubDirs {
		err = integrateBinFilesToSrc(filepath.Join(wsObjects, subDir))
		if err != nil {
			return err
		}
	}

	// first integrate bin parts into source files before adding BOM to it. Is simpler since bin integration has been
	// implemented already without BOM.
	err = ConvertDirectoryContentToUTF8Bom(wsObjects)
	if err != nil {
		return fmt.Errorf("failed to convert ws_objects content to UTF-8 bom: %v", err)
	}
	return nil
}

// integrateBinFilesToSrc reads all files in ws_objects source dir and if there are .bin files, those are integrated
// into their equally named .sr* file.
func integrateBinFilesToSrc(srcDir string) error {
	// for bin file -> read content -> add it to src file
	fmt.Println("create ws_objects source dir:", srcDir)
	binFiles := make(map[string]bool)
	var foundSrcFiles []string
	err := filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if filepath.Ext(path) == ".bin" {
			binFiles[strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))] = true
		} else if filepath.Ext(path) == ".pblmeta" {
			return os.Remove(path)
		} else if !d.IsDir() {
			foundSrcFiles = append(foundSrcFiles, path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	fmt.Println("Found ", len(foundSrcFiles), " files")
	fmt.Println("Found ", len(binFiles), " binaries")

	if len(binFiles) > 0 {
		return integrateBinToSrc(foundSrcFiles, binFiles)
	}
	return nil
}

func integrateBinToSrc(foundSrcFiles []string, binFiles map[string]bool) error {
	for _, file := range foundSrcFiles {
		if _, hasBin := binFiles[strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))]; hasBin {
			binFile := strings.TrimSuffix(file, filepath.Ext(file)) + ".bin"
			binSection, err := importer.GetBinarySectionFromBin(binFile)
			if err != nil {
				return fmt.Errorf("failed to set OLE binary section to matching bin file %s: %v",
					binFile, err)
			}
			f, err := os.OpenFile(file, os.O_RDWR|os.O_APPEND, 0o666)
			if err != nil {
				return fmt.Errorf("failed to open file %s: %v", file, err)
			}
			_, err = f.Write(binSection)
			_ = f.Close()
			if err != nil {
				return fmt.Errorf("failed to add bin section to %s: %v", file, err)
			}
			err = os.Remove(binFile)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func ConvertDirectoryContentToUTF8Bom(rootPath string) error {
	info, err := os.Stat(rootPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("directory %s does not exist: %v", rootPath, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("provided path is not a directory: %s", rootPath)
	}
	err = filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if err := processFile(path); err != nil {
			return fmt.Errorf("failed to process file %s: %v", path, err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// processFile converts the file if not yet to UTF-8 BOM encoding and adds export heade
func processFile(path string) error {
	srcFile, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("could not open source file: %v", err)
	}
	defer srcFile.Close()
	header := make([]byte, 3)
	n, err := io.ReadFull(srcFile, header)
	if err != nil && err != io.EOF && !errors.Is(err, io.ErrUnexpectedEOF) {
		return fmt.Errorf("could not read file header: %v", err)
	}

	actualHeader := header[:n]
	if bytes.Equal(actualHeader, utf8BOM) {
		fmt.Printf("Skip %s (already has BOM)\n", path)
		return nil
	}

	info, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("could not get file stats: %v", err)
	}
	tmpFile, err := os.CreateTemp(filepath.Dir(path), "bom-")
	if err != nil {
		return fmt.Errorf("could not create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()
	if _, err := tmpFile.Write(utf8BOM); err != nil {
		return fmt.Errorf("could not write BOM to temp file: %v", err)
	}
	if _, err := tmpFile.Write([]byte(fmt.Sprintf("$PBExportHeader$%s\r\n", filepath.Base(path)))); err != nil {
		return fmt.Errorf("could not write export header to temp file: %v", err)
	}
	if _, err := tmpFile.Write(actualHeader); err != nil {
		return fmt.Errorf("could not write header to temp file: %v", err)
	}
	if _, err := io.Copy(tmpFile, srcFile); err != nil {
		return fmt.Errorf("could not copy file content: %v", err)
	}
	srcFile.Close()
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("could not close temp file: %v", err)
	}
	if err := os.Chmod(tmpFile.Name(), info.Mode()); err != nil {
		return fmt.Errorf("could not set permissions on temp file: %v", err)
	}
	if err := os.Rename(tmpFile.Name(), path); err != nil {
		return fmt.Errorf("could not rename temp file: %v", err)
	}
	return nil
}
