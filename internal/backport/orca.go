package backport

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var orcaCreateAppPbl = `start session
scc set connect property localprojpath "."
scc connect offline
scc set target "demo.pbt" importonly
scc refresh target migrate
scc close
end session
`

// CreateApplicationPbl creates a fresh and minimal application pbl that can then be used to import actual
// application content. targetName is e.g. loh, appEntryPbl is e.g. loh1
// TODO check if this is also possible without exec cmd but using lib.go.base.pborca
func CreateApplicationPbl(targetName, appEntryPbl, operationDir string) (pblFile string, err error) {
	orcaExe, err := exec.LookPath("orcascr220.exe")
	if err != nil {
		return "",
			fmt.Errorf("orcascr220.exe is not within PATH, but needed to create a new application PBL")
	}
	tempOrcaScript, err := os.CreateTemp(operationDir, "create_app_pbl_*.orca")
	if err != nil {
		return "", fmt.Errorf("failedd to create temp orca script to create app pbl: %v", err)
	}
	defer tempOrcaScript.Close()
	defer os.Remove(tempOrcaScript.Name())
	_, err = tempOrcaScript.WriteString(strings.Replace(orcaCreateAppPbl, "demo", targetName, -1))
	if err != nil {
		return "", fmt.Errorf("failed to write orca script content to %s: %v", tempOrcaScript.Name(), err)
	}
	cmd := exec.Command(orcaExe, tempOrcaScript.Name())
	cmd.Dir = operationDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("cmd %q failed: %v; output: %s", cmd.String(), err, output)
	}
	appNamePbl := filepath.Join(operationDir, appEntryPbl+".pbl")
	_, err = os.Stat(appNamePbl)
	if err != nil {
		return "", fmt.Errorf("%s creation failed, file not present at %s", appNamePbl, operationDir)
	}
	return appNamePbl, err
}
