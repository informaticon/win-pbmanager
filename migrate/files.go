package migrate

import (
	"archive/zip"
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/informaticon/dev.win.base.pbmanager/utils"
	pborca "github.com/informaticon/lib.go.base.pborca"
	"github.com/informaticon/lib.go.base.pborca/orca"
)

//go:embed oldFiles.txt
var oldFiles string

var urlPbdk = "https://choco.informaticon.com/endpoints/axp/content/lib.bin.base.pbdk@22.2.0-3289.zip"
var urlPbdom = "https://choco.informaticon.com/endpoints/axp/content/lib.bin.base.pbdom@22.2.0-3289.pbl"

func RemoveFiles(folder string, warnFunc func(string)) error {
	lines := strings.Split(string(oldFiles), "\r\n")
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		if line[:1] == ";" {
			continue
		}
		if !utils.FileExists(filepath.Join(folder, line)) {
			continue
		}

		err := os.Remove(filepath.Join(folder, line))
		if err != nil {
			return fmt.Errorf("RemoveFiles failed: %v", err)
		}
	}
	err := utils.RemoveGlob(fmt.Sprintf("%s/*.*", filepath.Join(folder, "pbdk")))
	if err != nil {
		return fmt.Errorf("RemoveFiles failed: %v", err)
	}
	return nil
}

func FixPbInit(folder string, warnFunc func(string)) error {
	// read in pb.ini
	// if it does not exist or accessibility is not set
	// the function is aborted without error => nothing to migrate
	file := filepath.Join(folder, "pb.ini")
	if !utils.FileExists(file) {
		return nil
	}
	src, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("FixPbInit failed: %v", err)
	}

	// Comment out Accessibility setting
	regex := regexp.MustCompile(`(?mi)(\[Data Window\][^[]*?[\r\n]+)(Accessibility[ ]*=[ ]*[01][ ]*)[\r\n]`)
	valAccessibility := regex.FindSubmatch(src)
	if len(valAccessibility) == 0 {
		warnFunc(fmt.Sprintf("Unexpected content in pb.ini in folder %s", folder))
		return nil
	}
	src = regex.ReplaceAll(src, []byte("$1; Migrated by BSW-DEV\r\n; $2\r\n"))

	src = append(src, []byte("\r\n\r\n")...)

	regex = regexp.MustCompile(`(?mi)(\[Application\][^[]*)`)
	valApplication := regex.Find(src)
	if len(valApplication) > 0 {
		// Move [Application] to the end of the file
		src = regex.ReplaceAll(src, []byte{})
		src = append(src, valApplication...)
	} else {
		src = append(src, []byte("[Application]\r\n")...)
	}
	src = append(src, valAccessibility[2]...)

	err = os.WriteFile(file, src, 0664)
	if err != nil {
		return fmt.Errorf("FixPbInit failed: %v", err)
	}
	return nil
}

func InsertNewPbdk(libFolder string) error {
	pbdkZipFile, err := utils.GetRessource(urlPbdk)
	if err != nil {
		return fmt.Errorf("InsertNewPbdk failed while downloading pbdk: %v", err)
	}
	pbdkZip, err := zip.OpenReader(pbdkZipFile)
	if err != nil {
		return fmt.Errorf("InsertNewPbdk failed while opening zip file: %v", err)
	}
	defer pbdkZip.Close()

	for _, srcFSObj := range pbdkZip.File {
		dstPath := filepath.Join(libFolder, "pbdk", srcFSObj.Name)
		if srcFSObj.FileInfo().IsDir() {
			os.MkdirAll(dstPath, os.ModePerm)
			continue
		}
		err := os.MkdirAll(filepath.Dir(dstPath), os.ModePerm)
		if err != nil {
			return fmt.Errorf("InsertNewPbdk failed while creating dir: %v", err)
		}
		dstFile, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcFSObj.Mode())
		if err != nil {
			return fmt.Errorf("InsertNewPbdk failed while opening dst file: %v", err)
		}
		defer dstFile.Close()

		srcFile, err := srcFSObj.Open()
		if err != nil {
			return fmt.Errorf("InsertNewPbdk failed while reading zip file: %v", err)
		}
		defer srcFile.Close()

		_, err = io.Copy(dstFile, srcFile)
		if err != nil {
			return fmt.Errorf("InsertNewPbdk failed while copying file: %v", err)
		}
	}

	return nil
}

func InsertNewPbdom(libFolder string, appName string) error {
	pbdomFile, err := utils.GetRessource(urlPbdom)
	if err != nil {
		return fmt.Errorf("InsertNewPbdom failed: %v", err)
	}
	dstFileName := filepath.Join(libFolder, "pbdom.pbl")
	err = utils.CopyFile(pbdomFile, dstFileName)
	if err != nil {
		return fmt.Errorf("InsertNewPbdom failed: %v", err)
	}

	pbtFilePath := filepath.Join(libFolder, appName+".pbt")
	pbtData, err := os.ReadFile(pbtFilePath)
	if err != nil {
		return fmt.Errorf("InsertNewPbdom failed: %v", err)
	}

	// remove old pbdom
	pbtData = regexp.MustCompile(`(?mi);pbdom[0-9]{0,3}\.(?:pbl|pbd)`).ReplaceAll(pbtData, []byte{})
	// add new pbdom
	pbtData = regexp.MustCompile(`(?mi)^(LibList[ \t]+".*?)";`).ReplaceAll(pbtData, []byte(`$1;pbdom.pbl";`))

	err = os.WriteFile(pbtFilePath, pbtData, 0664)
	if err != nil {
		return fmt.Errorf("InsertNewPbdom failed: %v", err)
	}
	return nil
}

// InsertExfInPbt adds exf1.pbl to the library list, if it's needed
func InsertExfInPbt(pbtData *orca.Pbt, orca *pborca.Orca) error {
	src, err := orca.GetObjSource(pbtData.AppLib, pbtData.AppName+".sra")
	if err != nil {
		return nil
	}
	if !regexp.MustCompile("u_exf_error_manager[ \t]+gu_e").MatchString(src) {
		//no exf needed
		return nil
	}
	if slices.Contains(pbtData.LibList, filepath.Join(pbtData.BasePath, "exf1.pbl")) {
		//exf already in library list
		return nil
	}

	// Fix lib list in current Data obj
	pbtData.LibList = append(pbtData.LibList, filepath.Join(pbtData.BasePath, "exf1.pbl"))

	// Fix lib list in pbt file
	pbtFile, err := os.ReadFile(filepath.Join(pbtData.BasePath, pbtData.AppName+".pbt"))
	if err != nil {
		return fmt.Errorf("InsertExfInPbt failed: %v", err)
	}

	// add new pbdom
	pbtFile = regexp.MustCompile(`(?mi)^(LibList[ \t]+".*?)(;inf3.pbl;.*?")`).ReplaceAll(pbtFile, []byte(`${1};exf1.pbl${2}`))

	err = os.WriteFile(filepath.Join(pbtData.BasePath, pbtData.AppName+".pbt"), pbtFile, 0664)
	if err != nil {
		return fmt.Errorf("InsertExfInPbt failed: %v", err)
	}
	return nil
}
