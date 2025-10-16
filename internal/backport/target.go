package backport

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Target contains basic components to create a .pbt file. There might be others that are not necessary for compilation.
type Target struct {
	AppName string
	AppLib  string
	LibList []string
}

// NewTarget returns the minimal structure of needed for a target file. Expects a list of pbl names, e.g.
// []string{"some_app.pbl", "some_lib.pbl", ...}
func NewTarget(appName, appEntryPbl string, libList []string) Target {
	return Target{
		AppName: appName,
		AppLib:  appEntryPbl,
		LibList: libList,
	}
}

// ToBytes must cut off the library list paths and can only consider the file name, e.g. not test\\exf1.pbl but only
// exf1.pbl.
// This is needed since else the backporting is not working properly as pbautobuild220.exe expects a flat pbl structure
// within ws_objects and beside the target. Else an error "application not set" occurs.
func (t Target) ToBytes() []byte {
	for i, lib := range t.LibList {
		t.LibList[i] = filepath.Base(lib)
	}
	t.AppLib = filepath.Base(t.AppLib)

	targetString := fmt.Sprintf(`Save Format v3.0(19990112)
appname "%s";
applib "%s";
liblist "%s";
type "pb";`,
		t.AppName,
		strings.ReplaceAll(filepath.ToSlash(t.AppLib), "/", "\\\\"),
		strings.ReplaceAll(strings.ReplaceAll(strings.Join(t.LibList, ";"), "\\", "/"), "/", "\\\\"),
	)
	return []byte(targetString)
}
