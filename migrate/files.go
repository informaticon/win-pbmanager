package migrate

import (
	"archive/zip"
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/informaticon/dev.win.base.pbmanager/utils"
)

//go:embed oldFiles.txt
var oldFiles string

var urlPbdk = "https://choco.informaticon.com/endpoints/axp/content/lib.bin.base.pbdk@22.2.0-3238.zip"
var urlPbdom = "https://choco.informaticon.com/endpoints/axp/content/lib.bin.base.pbdom@22.2.0-3238.pbd"

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
			return err
		}
	}
	return utils.RemoveGlob(fmt.Sprintf("%s/*.*", filepath.Join(folder, "pbdk")))
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
		return err
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

	return os.WriteFile(file, src, 0664)
}

func InsertNewPbdk(libFolder string) error {
	pbdkZipFile, err := utils.GetRessource(urlPbdk)
	if err != nil {
		return err
	}
	pbdkZip, err := zip.OpenReader(pbdkZipFile)
	if err != nil {
		return err
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
			return err
		}
		dstFile, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcFSObj.Mode())
		if err != nil {
			return err
		}
		defer dstFile.Close()

		srcFile, err := srcFSObj.Open()
		if err != nil {
			return err
		}
		defer srcFile.Close()

		_, err = io.Copy(dstFile, srcFile)
		if err != nil {
			return err
		}
	}

	return nil
}

func InsertNewPbdom(libFolder string) error {
	pbdomFile, err := utils.GetRessource(urlPbdom)
	if err != nil {
		return err
	}
	dstFileName := filepath.Join(libFolder, "pbdom220.pbd")
	err = utils.CopyFile(pbdomFile, dstFileName)
	if err != nil {
		return err
	}

	pbtFilePath := filepath.Join(libFolder, "a3.pbt")
	pbtData, err := os.ReadFile(pbtFilePath)
	if err != nil {
		return err
	}

	// remove old pbdom
	pbtData = regexp.MustCompile(`(?mi);pbdom[0-9]{3}\.(?:pbl|pbd)`).ReplaceAll(pbtData, []byte{})
	// add new pbdom
	pbtData = regexp.MustCompile(`(?mi)^(LibList[ \t]+".*?)";`).ReplaceAll(pbtData, []byte(`$1;pbdom220.pbd";`))

	return os.WriteFile(pbtFilePath, pbtData, 0664)
}
