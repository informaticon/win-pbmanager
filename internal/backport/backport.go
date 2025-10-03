// Package backport expects a solution file and for each listed project a target and related libraries are created
// so that the target can be compiled with PB2022.
package backport

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ConvertProjectToTarget modifies src files referenced by .pbproj directory and converts the project back to target.
func ConvertProjectToTarget(pbProjFile string) error {
	rules := []FileRule{
		{Matcher: matchExt(".srd"), Handler: handleSrdFile},
		{Matcher: matchExt(".sra"), Handler: handleSraFile},
	}
	err := ConvertSrcDirs([]string{filepath.Dir(pbProjFile)}, rules)
	if err != nil {
		return err
	}
	pbProj, err := NewProject(pbProjFile)
	if err != nil {
		return err
	}

	// Create pbt file
	pbtFilePath := filepath.Join(filepath.Dir(pbProjFile), strings.TrimSuffix(filepath.Base(pbProjFile), ".pbproj")+".pbt")
	err = os.WriteFile(pbtFilePath,
		NewTarget(pbProj.Application.Name, pbProj.Libraries.AppEntry, pbProj.Libraries.GetPblPaths()).ToBytes(),
		0o644)
	if err != nil {
		return fmt.Errorf("failed to write actual application target %s: %v", pbtFilePath, err)
	}

	err = Src25ToWsObjects(pbProj)
	if err != nil {
		return err
	}

	jsonTemplate := []byte(fmt.Sprintf(`{
    "MetaInfo": {
        "IDEVersion": "220",
        "RuntimeVersion": "22.2.0.3356"
    },
    "BuildPlan": {
        "SourceControl": {
            "Merging": [
                {"Target": ".\\%[1]s.pbt", "LocalProjectPath": ".", "RefreshPbl": true}
            ]
        },
        "BuildJob": {
            "Projects": [
                {"Target": ".\\%[1]s.pbt","Name": "%[1]s"}
            ]
        }
    }
}`, strings.TrimSuffix(filepath.Base(pbProjFile), ".pbproj")))

	autoBuildJsonFile := filepath.Join(filepath.Dir(pbProjFile),
		strings.TrimSuffix(filepath.Base(pbProjFile), ".pbproj")+".json")
	err = os.WriteFile(autoBuildJsonFile, jsonTemplate, 0o644)
	if err != nil {
		return err
	}
	return runPbAutoBuild(autoBuildJsonFile)
}

// runPbAutoBuild executes the pbautobuild command with a given JSON config file
func runPbAutoBuild(jsonConfigPath string) error {
	cmd := exec.Command("pbautobuild220.exe", "/f", jsonConfigPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Println("run", cmd.String())
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start pbautobuild command %s: %v", cmd.String(), err)
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("pbautobuild (%s) execution failed: %v", cmd.String(), err)
	}
	return nil
}
