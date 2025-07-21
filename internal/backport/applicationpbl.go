package backport

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// myAppPbg template for .pbg file. First part is entry lib name, second arg is appname, e.g.
// $PBExportHeader$loh1.pbg
// loh.sra
const myAppPbg = `$PBExportHeader$%s.pbg
%s.sra`

// myAppSra is a template application source: Replace 'demo' with actual app name. Is base64 encoded since every byte
// matters (e.g. CRLF).
//
//go:embed template/demo.sra
var myAppSra []byte

// GetApplicationPbl creates a pseudo target, group and application source to generate via an orca cmd a
// new minimal application PBL with its actual name. One can not recycle any application pbl since it contains
// offset bytes depending on the name, i.e. only replacing the name bytes would not be enough.
func GetApplicationPbl(appName, appEntryPbl, destinationDir string) error {
	appEntryPblName := filepath.Base(appEntryPbl)[:len(filepath.Base(appEntryPbl))-len(filepath.Ext(appEntryPbl))]
	tempDir, err := os.MkdirTemp("", "appLibCreation-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)
	err = os.WriteFile(filepath.Join(tempDir, fmt.Sprintf("%s.pbg", appName)),
		[]byte(fmt.Sprintf(myAppPbg, appEntryPblName, appName)), 0644)
	if err != nil {
		return fmt.Errorf("failed to create temp .pbg file: %v", err)
	}
	err = os.WriteFile(filepath.Join(tempDir, fmt.Sprintf("%s.sra", appName)),
		bytes.ReplaceAll(myAppSra, []byte("demo"), []byte(appName)), 0644)
	if err != nil {
		return fmt.Errorf("failed to create temp .sra file: %v", err)
	}
	err = os.WriteFile(filepath.Join(tempDir, fmt.Sprintf("%s.pbt", appName)),
		NewTarget(appName, appEntryPbl, []string{appEntryPblName + ".pbl"}).ToBytes(), 0644)
	if err != nil {
		return fmt.Errorf("failed to create temp .pbt file: %v", err)
	}
	// creates appname.pbl different from empty.pbl
	appPblPath, err := CreateApplicationPbl(appName, appEntryPblName, tempDir)
	if err != nil {
		return fmt.Errorf("failed to create temp application .pbl file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(destinationDir, filepath.Base(appPblPath))); errors.Is(err, os.ErrNotExist) {
		errRename := os.Rename(appPblPath, filepath.Join(destinationDir, filepath.Base(appPblPath)))
		if errRename != nil {
			return fmt.Errorf("failed to move %s to destination dir %s: %v",
				appPblPath, destinationDir, errRename)
		}
	}
	return nil
}
