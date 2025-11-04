// Package backport expects a solution file and for each listed project a target and related libraries are created
// so that the target can be compiled with PB2022.
package backport

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ConvertProjectToTarget modifies src files referenced by .pbproj directory and converts the project back to target.
func ConvertProjectToTarget(pbProjFile string, verbose bool) error {
	rules := []FileRule{
		{description: "FixDWHeader", Matcher: matchExt(".srd"), Handler: handleSrdFile},
		{description: "FixSraRuntime", Matcher: matchExt(".sra"), Handler: handleSraFile},
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

	err = Src25ToWsObjects(pbProj, verbose)
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
                {"Target": ".\\%[1]s.pbt","Name": "%[2]s"}
            ]
        }
    }
}`, strings.TrimSuffix(filepath.Base(pbProjFile), ".pbproj"), pbProj.Application.Name))

	autoBuildJsonFile := filepath.Join(filepath.Dir(pbProjFile),
		strings.TrimSuffix(filepath.Base(pbProjFile), ".pbproj")+".json")
	err = os.WriteFile(autoBuildJsonFile, jsonTemplate, 0o644)
	if err != nil {
		return err
	}
	return runPbAutoBuild(strings.TrimSuffix(filepath.Base(pbProjFile), ".pbproj"), verbose)
}

// runPbAutoBuild executes the pbautobuild command with ./a3.json!!!
// Must be exactly like this, no absolute path no, no "a3.json" -_-
func runPbAutoBuild(jsonName string, verbose bool) error {
	cmd := exec.Command("pbautobuild220.exe", "/f", fmt.Sprintf(".\\%s.json", jsonName))
	fmt.Printf("running command: %s\n", cmd.String())
	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("command '%s' failed: %w\n  stdout: %s\n  stderr: %s",
			cmd.String(),
			err,
			strings.TrimSpace(stdoutBuf.String()),
			strings.TrimSpace(stderrBuf.String()),
		)
	}
	return nil
}
