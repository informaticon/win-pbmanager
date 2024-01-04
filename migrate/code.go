package migrate

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/informaticon/dev.win.base.pbmanager/utils"
	pborca "github.com/informaticon/lib.go.base.pborca"
	"github.com/informaticon/lib.go.base.pborca/orca"
)

func FixRegistry(libFolder string, targetName string, orca *pborca.Orca, warnFunc func(string)) error {
	pblFile := filepath.Join(libFolder, "inf1.pbl")
	pbtFile := filepath.Join(libFolder, targetName+".pbt")
	objName := "inf1_u_registry"
	regex := regexp.MustCompile(`(?mi)[ ]*string[ ]+is_ie_ole_exes\[\][ ]*=[ ]*\{(.*)\}[ ]*`)

	src, err := orca.GetObjSource(pblFile, objName)
	if err != nil {
		return err
	}
	matches := regex.FindAllStringSubmatch(src, -1)
	if len(matches) != 1 {
		return fmt.Errorf("ole string is not present in project %s", libFolder)
	}
	if strings.Trim(matches[0][1], " ") != `"a3.exe", "pb170.exe"` &&
		strings.Trim(matches[0][1], " ") != `"a3.exe", "pb170.exe", "pb220.exe", "pb250.exe"` {
		warnFunc(fmt.Sprintf("%s in folder %s doesnt contain the expected content (%s)", objName, libFolder, matches[0][1]))
	}
	src = regex.ReplaceAllString(src, `string is_ie_ole_exes[] = {"a3.exe", "pb170.exe", "pb220.exe", "pb250.exe"}`)

	err = orca.SetObjSource(pbtFile, pblFile, objName, src)
	if err != nil {
		return err
	}

	return nil
}

func FixLibInterface(libFolder string, targetName string, orca *pborca.Orca, warnFunc func(string)) error {
	pblFile := filepath.Join(libFolder, "lif1.pbl")
	pbtFile := filepath.Join(libFolder, targetName+".pbt")
	objName := "lif1_u_process"
	//if lower(ls_exe) = "pb115.exe" or lower(ls_exe) = "pb170.exe" then
	regex := regexp.MustCompile(`(?mi)[ \t]*if[ ]+(lower\([ ]*ls_exe[ ]*\)[ ]*=[ ]*"pb[0-9]{3}\.exe".*?)then[ ]*`)

	src, err := orca.GetObjSource(pblFile, objName)
	if err != nil {
		return err
	}
	matches := regex.FindAllStringSubmatch(src, -1)
	if len(matches) != 1 {
		return fmt.Errorf("exe string is not present in project %s", libFolder)
	}
	if strings.Trim(matches[0][1], " ") != `lower(ls_exe) = "pb115.exe" or lower(ls_exe) = "pb170.exe"` &&
		strings.Trim(matches[0][1], " ") != `lower(ls_exe) = "pb170.exe" or lower(ls_exe) = "pb220.exe" or lower(ls_exe) = "pb250.exe"` {
		warnFunc(fmt.Sprintf("%s in folder %s doesnt contain the expected content (%s)", objName, libFolder, matches[0][1]))
	}
	src = regex.ReplaceAllString(src, `	if lower(ls_exe) = "pb170.exe" or lower(ls_exe) = "pb220.exe" or lower(ls_exe) = "pb250.exe" then`)

	err = orca.SetObjSource(pbtFile, pblFile, objName, src)
	if err != nil {
		return err
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
		return err
	}
	for _, file := range files {
		objSrc, err := mirrorFiles.ReadFile("mirror_objects/" + file.Name())
		if err != nil {
			return err
		}
		objName := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))

		err = orca.SetObjSource(pbtFile, pblFile, objName, string(objSrc))
		if err != nil {
			return err
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
			return err
		}
		src = regexp.MustCompile(`(?mi)^(EXE:.*?)[A-Z]:\\[^,]+$`).ReplaceAllString(src, "$1.\\pbdk\r\n")
		err = orca.SetObjSource(pbtFile, pblFile, proj.Name, src)
		if err != nil {
			return err
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
		return err
	}
	regr := regexp.MustCompile(`(?mi)(@begin Projects[^@]*?&` + projName + `&)` + oldLib + `(";[^@]*?@end;)`)
	pbtData = regr.ReplaceAll(pbtData, []byte("${1}"+newLib+"${2}"))
	return os.WriteFile(pbtFilePath, pbtData, 0664)
}
