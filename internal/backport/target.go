package backport

import (
	"fmt"
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

func (t Target) ToBytes() []byte {
	targetString := fmt.Sprintf(`Save Format v3.0(19990112)
appname "%s";
applib "%s";
liblist "%s";
type "pb";`, t.AppName, t.AppLib, strings.Join(t.LibList, ";"))
	return []byte(targetString)
}
