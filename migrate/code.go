package migrate

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	_ "embed"

	"github.com/informaticon/dev.win.base.pbmanager/utils"
	pborca "github.com/informaticon/lib.go.base.pborca"
	"github.com/informaticon/lib.go.base.pborca/orca"
)

//go:embed pb_files/a3_lohn.pbw
var pbwA3Lohn []byte

func FixRegistry(libFolder string, targetName string, orca *pborca.Orca, warnFunc func(string)) error {
	pblFile := filepath.Join(libFolder, "inf1.pbl")
	pbtFile := filepath.Join(libFolder, targetName+".pbt")
	objName := "inf1_u_registry"
	regex := regexp.MustCompile(`(?mi)[ ]*string[ ]+is_ie_ole_exes\[\][ ]*=[ ]*\{(.*)\}[ ]*`)

	src, err := orca.GetObjSource(pblFile, objName)
	if err != nil {
		warnFunc(fmt.Sprintf("skipping inf1_u_registry migration (file %s does not contain an object named %s)", pblFile, objName))
		return nil
	}
	matches := regex.FindAllStringSubmatch(src, -1)
	if len(matches) != 1 {
		return fmt.Errorf("FixRegistry failed: ole string is not present in project %s", libFolder)
	}
	if strings.Trim(matches[0][1], " ") != `"a3.exe", "pb170.exe"` &&
		strings.Trim(matches[0][1], " ") != `"a3.exe", "pb170.exe", "pb220.exe", "pb250.exe"` {
		warnFunc(fmt.Sprintf("  %s in file %s doesnt contain the expected content (%s)", objName, libFolder, matches[0][1]))
	}
	src = regex.ReplaceAllString(src, `string is_ie_ole_exes[] = {"a3.exe", "pb170.exe", "pb220.exe", "pb250.exe"}`)

	err = orca.SetObjSource(pbtFile, pblFile, objName, src)
	if err != nil {
		return fmt.Errorf("FixRegistry failed: %v", err)
	}

	return nil
}

func FixLifProcess(libFolder string, targetName string, orca *pborca.Orca, warnFunc func(string)) error {
	pblFile := filepath.Join(libFolder, "lif1.pbl")
	pbtFile := filepath.Join(libFolder, targetName+".pbt")

	objName := "lif1_u_process"
	src, err := orca.GetObjSource(pblFile, objName)
	if err != nil {
		objName = "inf1_u_process"
		src, err = orca.GetObjSource(pblFile, objName)
		if err != nil {
			fmt.Printf("skipping %s migration (%v)\n", objName, err)
			return nil
		}
	}

	//if lower(ls_exe) = "pb115.exe" or lower(ls_exe) = "pb170.exe" then
	regex := regexp.MustCompile(`(?mi)[ \t]*if[ ]+(lower\([ ]*ls_exe[ ]*\)[ ]*=[ ]*"pb[0-9]{3}\.exe".*?)then[ ]*`)

	matches := regex.FindAllStringSubmatch(src, -1)
	if len(matches) != 1 {
		return fmt.Errorf("FixLifProcess failed: exe string is not present in project %s", libFolder)
	}
	if strings.Trim(matches[0][1], " ") != `lower(ls_exe) = "pb115.exe" or lower(ls_exe) = "pb170.exe"` &&
		strings.Trim(matches[0][1], " ") != `lower(ls_exe) = "pb170.exe" or lower(ls_exe) = "pb220.exe" or lower(ls_exe) = "pb250.exe"` {
		warnFunc(fmt.Sprintf("  %s in folder %s doesnt contain the expected content (%s)", objName, libFolder, matches[0][1]))
	}
	src = regex.ReplaceAllString(src, `	if lower(ls_exe) = "pb170.exe" or lower(ls_exe) = "pb220.exe" or lower(ls_exe) = "pb250.exe" then`)
	err = orca.SetObjSource(pbtFile, pblFile, objName, src)
	if err != nil {
		return fmt.Errorf("FixLifProcess failed: %v", err)
	}
	return nil
}

// PB115 migration: Replace _DEBUG with CI_DEBUG...
func FixLifMetratec(libFolder string, targetName string, orca *pborca.Orca, warnFunc func(string), ignoreCompileErr bool) error {
	pblFile := filepath.Join(libFolder, "lif1.pbl")
	pbtFile := filepath.Join(libFolder, targetName+".pbt")

	objName := "lif1_u_metratec_base"
	src, err := orca.GetObjSource(pblFile, objName)
	if err != nil {
		objName := "inf1_u_metratec_base"
		src, err = orca.GetObjSource(pblFile, objName)
		if err != nil {
			return fmt.Errorf("FixLifMetratec failed: %v", err)
		}
	}

	regex := regexp.MustCompile(`(?im)([ \t])(_INFO|_FATAL|_ERROR|_DEBUG|_WARN)`)
	src = regex.ReplaceAllString(src, `${1}CI${2}`)
	err = orca.SetObjSource(pbtFile, pblFile, objName, src)
	if err != nil && !ignoreCompileErr {
		return fmt.Errorf("FixLifMetratec failed: %v", err)
	}
	return nil
}

// FixLohXmlDecl removes deprecated use of pbdom_processinginstruction
func FixLohXmlDecl(libFolder string, targetName string, orca *pborca.Orca, warnFunc func(string)) error {
	pbtFile := filepath.Join(libFolder, targetName+".pbt")
	regex1 := regexp.MustCompile(`(?im)lpbdom_pi.setname\('xml'\)[\r\n\t ]+lpbdom_pi.SetData\('version="1\.0" encoding="UTF-8"'\)[\t ]+`)
	regex2 := regexp.MustCompile(`(?im)[\r\n\t ]+ipbdom_document.addcontent\(lpbdom_pi\)[\t ]+`)
	for objName, pblFile := range map[string]string{
		"loh1_u_loh_xml_salary_declaration": filepath.Join(libFolder, "loh1.pbl"),
		"elm1_u_elm_xml_salary_declaration": filepath.Join(libFolder, "elmg.pbl"),
	} {
		src, err := orca.GetObjSource(pblFile, objName)
		if err != nil {
			warnFunc(fmt.Sprintf("skipping %s migration (does not exist in %s)", objName, pblFile))
			continue
		}
		src = regex1.ReplaceAllString(src, `ipbdom_document.setxmldeclaration("1.0", "UTF-8", "yes")`)
		src = regex2.ReplaceAllString(src, ``)

		err = orca.SetObjSource(pbtFile, pblFile, objName, src)
		if err != nil {
			return fmt.Errorf("FixLohXmlDecl failed on %s: %v", objName, err)
		}
	}
	return nil
}

//go:embed mirror_objects/*.sr*
var mirrorFiles embed.FS

func AddMirrorObjects(libFolder string, targetName string, orca *pborca.Orca, warnFunc func(string)) error {
	pblFile := filepath.Join(libFolder, "inf1.pbl")
	pbtFile := filepath.Join(libFolder, targetName+".pbt")
	files, err := mirrorFiles.ReadDir("mirror_objects")
	if err != nil {
		return fmt.Errorf("AddMirrorObjects failed: %v", err)
	}
	for _, file := range files {
		objSrc, err := mirrorFiles.ReadFile("mirror_objects/" + file.Name())
		if err != nil {
			return fmt.Errorf("AddMirrorObjects failed: %v", err)
		}
		objName := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))

		err = orca.SetObjSource(pbtFile, pblFile, objName, string(objSrc))
		if err != nil {
			return fmt.Errorf("AddMirrorObjects failed: %v", err)
		}
	}
	return nil
}

func FixRuntimeFolder(pbtData *orca.Pbt, orca *pborca.Orca, warnFunc func(string)) error {
	// In non-a3 projects, there may be no pbdk folder
	if !utils.FileExists(filepath.Join(pbtData.BasePath, "pbdk")) {
		return nil
	}

	pbtFile := filepath.Join(pbtData.BasePath, pbtData.AppName+".pbt")
	for _, proj := range pbtData.Projects {
		pblFile := filepath.Join(pbtData.BasePath, proj.PblFile)
		src, err := orca.GetObjSource(pblFile, proj.Name+".srj")
		if err != nil {
			return fmt.Errorf("FixRuntimeFolder failed: %v", err)
		}
		src = regexp.MustCompile(`(?mi)^(EXE:.*?)[A-Z]:\\[^,]+$`).ReplaceAllString(src, "$1.\\pbdk\r\n")
		err = orca.SetObjSource(pbtFile, pblFile, proj.Name, src)
		if err != nil {
			return fmt.Errorf("FixRuntimeFolder failed: %v", err)
		}
	}
	return nil
}

// FixProjLib replaces wrong project line in pbt file.
//
// For example, the Line `@begin Projects\n 0 "1&a3&inf2.pbl";\n@end;`
// can be replaced with `@begin Projects\n 0 "1&a3&inf1.pbl";\n@end;`
func FixProjLib(pbtFilePath, projName, oldLib, newLib string) error {
	pbtData, err := os.ReadFile(pbtFilePath)
	if err != nil {
		return fmt.Errorf("FixProjLib failed: %v", err)
	}
	regr := regexp.MustCompile(`(?mi)(@begin Projects[^@]*?&` + projName + `&)` + oldLib + `(";[^@]*?@end;)`)
	pbtData = regr.ReplaceAll(pbtData, []byte("${1}"+newLib+"${2}"))
	err = os.WriteFile(pbtFilePath, pbtData, 0664)
	if err != nil {
		return fmt.Errorf("FixProjLib failed: %v", err)
	}
	return nil
}

// ReplacePayrollPbwFile replaces the pbwFile (to get rid of other targets)
func ReplacePayrollPbwFile(pbwFilePath string) error {
	return os.WriteFile(pbwFilePath, pbwA3Lohn, 0664)
}
