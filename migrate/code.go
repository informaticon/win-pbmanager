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

// FixSqla17ByteString replaces all occurrances of byte_substr with substr
func FixSqla17ByteString(libFolder string, targetName string, orca *pborca.Orca, warnFunc func(string)) error {
	pbtFile := filepath.Join(libFolder, targetName+".pbt")
	regex := regexp.MustCompile(`(?is)([^a-z])byte_substr\(`)
	var fixedObjNames []string
	pblFiles, err := filepath.Glob(filepath.Join(libFolder, "*.pbl"))
	if err != nil {
		return fmt.Errorf("could not create list of pbl files for folder %s: %v", libFolder, err)
	}
	for _, pblFile := range pblFiles {
		objArrs, err := orca.GetObjList(pblFile)
		if err != nil {
			return fmt.Errorf("could not create list of objects for pbl %s: %v", pblFile, err)
		}

		for _, objArr := range objArrs {
			for _, obj := range objArr.GetObjArr() {
				objSrc, err := orca.GetObjSource(pblFile, obj.Name+pborca.GetObjSuffixFromType(obj.ObjType))
				if err != nil {
					return fmt.Errorf("could not get soure for object %s: %v", obj.Name+pborca.GetObjSuffixFromType(obj.ObjType), err)
				}
				if strings.Contains(objSrc, "byte_substr") {
					newObjSrc := regex.ReplaceAllString(objSrc, "${1}substr(")
					if newObjSrc == objSrc {
						warnFunc(fmt.Sprintf("found byte_substr pattern in %s but regex did not match it", obj.Name))
						continue
					}
					fixedObjNames = append(fixedObjNames, obj.Name)
					err = orca.SetObjSource(pbtFile, pblFile, obj.Name, []byte(newObjSrc))
					if err != nil {
						return fmt.Errorf("could not set source for object %s: %v", obj.Name, err)
					}
				}
			}
		}
	}
	fmt.Printf("FixSqla17 fixed %d objects: %v\n", len(fixedObjNames), fixedObjNames)
	return nil
}

func FixSqla17Base(libFolder string, targetName string, orca *pborca.Orca, warnFunc func(string)) error {
	pbtFile := filepath.Join(libFolder, targetName+".pbt")
	fixes := []struct {
		fixName string
		pblFile string
		objName string
		regex   *regexp.Regexp
		replace string
	}{
		{
			"FIX1", "inf1.pbl", "inf1_u_transaction.sru",
			regexp.MustCompile(`(?is)[\r\n](\/\/Version[\n\r\t ]+ls_version[\t =]+of_get_version\(\).*?end if)`),
			"\r\n//SQLA17 migration: deactivate driver check\r\n/*\r\n${1}\r\n*/",
		},
		{
			"FIX2", "inf1.pbl", "inf1_u_transaction.sru",
			regexp.MustCompile(`(?is)(([ \t]+if left\(ls_version,[ \t=]+2\)[ \t=]+'(11|16)'.*?[\r\n]+)+)`),
			"\r\n\t//SQLA17 migration: allow sqla17 driver\r\n\t/*\r\n${1}\t*/\r\n\tchoose case left(of_get_version(), 2)\r\n\t\tcase '11' //SQLA11\r\n\t\t\tas_db += \";commlinks=tcpip{host=\" + string(ls_host) + \"}\"\r\n\t\tcase else //SQLA16, SQLA17, ...\r\n\t\t\tas_db += \";host=\" + ls_host\r\n\tend choose\r\n",
		},
		{
			"FIX3", "inf1.pbl", "inf1_u_transaction.sru",
			regexp.MustCompile(`(?is)(public[ \t]+subroutine[ \t]+of_check_version[a-z_ \t()]+;)(.*?)([\r\n]+end[ t]+subroutine[\r\n]+)`),
			"${1}//SQLA17 migration: deactivate db version check\r\n/*OLD SOURCE HAS BEEN REMOVED*/\r\nlong ll_fun_exists\r\nstring ls_error\r\nselect count(*) into :ll_fun_exists from sysprocedure where proc_name = 'dev_check_sqla_versions';\r\nif ll_fun_exists = 0 then\r\n\t// function dev_check_sqla_versions does not exist, continue without db version check\r\n\treturn\r\nend if\r\n\r\nselect dev_check_sqla_versions() into :ls_error from dummy;\r\nif ls_error = '' then\r\n\treturn\r\nend if\r\n\r\nthrow(gu_e.iu_as.of_re_database(gu_e.of_new_error().of_push(populateerror(0, ls_error)).of_push('this', this)))${3}",
		},
	}
	for _, fix := range fixes {
		src, err := orca.GetObjSource(filepath.Join(libFolder, fix.pblFile), fix.objName)
		if err != nil {
			warnFunc(fmt.Sprintf("skipping fix %s, file %s does not contain an object named %s: %v", fix.fixName, fix.pblFile, fix.objName, err))
			continue
		}
		src = fix.regex.ReplaceAllString(src, fix.replace)
		err = orca.SetObjSource(pbtFile, filepath.Join(libFolder, fix.pblFile), fix.objName, []byte(src))
		if err != nil {
			return fmt.Errorf("fix %s for %s failed, could not write source: %v", fix.fixName, fix.objName, err)
		}

	}
	return nil
}

func FixArf(libFolder string, targetName string, orca *pborca.Orca, warnFunc func(string)) error {
	pblFile := filepath.Join(libFolder, "arf1.pbl")
	pbtFile := filepath.Join(libFolder, targetName+".pbt")
	objName := "arf1_u_arf_service_lohn"
	regex := regexp.MustCompile(`(?is)([ \t]*\/\/2020-10-02 Martin Abplanalp, Ticket 19529[^\r\n]+[\r\n\t ]+)(ls_release_liblohn.*?end if)`)

	src, err := orca.GetObjSource(pblFile, objName)
	if err != nil {
		warnFunc(fmt.Sprintf("skipping arf1_u_arf_service_lohn migration (file %s does not contain an object named %s)", pblFile, objName))
		return nil
	}
	matches := regex.FindAllStringSubmatch(src, -1)
	if len(matches) > 1 {
		return fmt.Errorf("FixArf failed: Too many matches: %v", matches)
	} else if len(matches) < 1 {
		warnFunc(fmt.Sprintf("skipping arf1_u_arf_service_lohn migration (file %s does not contain expected string)", pblFile))
		return nil
	}
	src = regex.ReplaceAllString(src, `${1}/*COMMENTED OUT BY PB2022R3 MIGRATION: ${2}*/`)

	err = orca.SetObjSource(pbtFile, pblFile, objName, []byte(src))
	if err != nil {
		return fmt.Errorf("FixArf failed: %v", err)
	}
	return nil
}

func FixRegistry(libFolder string, targetName string, orca *pborca.Orca, warnFunc func(string)) error {
	pblFile := filepath.Join(libFolder, "inf1.pbl")
	pbtFile := filepath.Join(libFolder, targetName+".pbt")
	objName := "inf1_u_registry"
	regex := regexp.MustCompile(`(?mi)[ ]*string[ ]+is_ie_ole_exes\[\][ ]*=[ ]*\{(.*)\}[ ]*`)

	src, err := orca.GetObjSource(pblFile, objName)
	if err != nil {
		warnFunc(fmt.Sprintf("skipping %s migration, file %s does not contain an object named %s", objName, pblFile, objName))
		return nil
	}

	matches := regex.FindAllStringSubmatch(src, -1)
	if len(matches) != 1 {
		return fmt.Errorf("FixRegistry failed: ole string is not present in project %s", libFolder)
	}
	if strings.Trim(matches[0][1], " ") == `"a3.exe", "pb170.exe", "pb220.exe", "pb250.exe"` {
		warnFunc(fmt.Sprintf("skipping %s migration, object %s is already migrated", objName, objName))
		return nil
	}
	if strings.Trim(matches[0][1], " ") != `"a3.exe", "pb170.exe"` &&
		strings.Trim(matches[0][1], " ") != `"a3.exe", "pb170.exe", "pb220.exe", "pb250.exe"` {
		warnFunc(fmt.Sprintf("  %s in file %s doesnt contain the expected content (%s)", objName, libFolder, matches[0][1]))
	}
	src = regex.ReplaceAllString(src, `string is_ie_ole_exes[] = {"a3.exe", "pb170.exe", "pb220.exe", "pb250.exe"}`)

	err = orca.SetObjSource(pbtFile, pblFile, objName, []byte(src))
	if err != nil {
		return fmt.Errorf("FixRegistry failed: %v", err)
	}

	return nil
}

func FixHttpClient(libFolder string, targetName string, orca *pborca.Orca, warnFunc func(string)) error {
	pbtFile := filepath.Join(libFolder, targetName+".pbt")

	// part 1: fin1_u_fin_bankenstamm
	step1 := func() error {
		pblFile := filepath.Join(libFolder, "fin1.pbl")
		objName := "fin1_u_fin_bankenstamm"
		regex := regexp.MustCompile(`(?mi)(lu_client = create httpclient)([\r\n \t]+)(li_ret)`)
		src, err := orca.GetObjSource(pblFile, objName)
		if err != nil {
			warnFunc(fmt.Sprintf("skipping %s migration, file %s doesn't contain %s", objName, pblFile, objName))
			return nil
		}
		if len(regex.FindAllString(src, -1)) == 0 {
			warnFunc(fmt.Sprintf("skipping %s migration, already migrated", objName))
			return nil
		}

		src = regex.ReplaceAllString(src, `${1}${2}lu_client.anonymousaccess = true${2}${3}`)

		err = orca.SetObjSource(pbtFile, pblFile, objName, []byte(src))
		if err != nil {
			return fmt.Errorf("FixHttpClient failed for %s: %v", objName, err)
		}
		return nil
	}
	err := step1()
	if err != nil {
		return err
	}

	// part 2: inf1_u_httpclient
	step2 := func() error {
		pblFile := filepath.Join(libFolder, "inf1.pbl")
		objName := "inf1_u_httpclient"
		src, err := orca.GetObjSource(pblFile, objName)
		if err != nil {
			warnFunc(fmt.Sprintf("skipping %s migration, file %s doesn't contain %s", objName, pblFile, objName))
			return nil
		}
		regex := regexp.MustCompile(`(?mi)global type inf1_u_httpclient from httpclient[\r\n \t]+boolean anonymousaccess = true`)
		if len(regex.FindAllString(src, -1)) >= 1 {
			warnFunc(fmt.Sprintf("skipping %s migration, already migrated", objName))
			return nil
		}

		regex = regexp.MustCompile(`(?mi)(end forward[\r\n\t ]+global type inf1_u_httpclient from httpclient[ \t]+)`)
		src = regex.ReplaceAllString(src, `${1}\r\nboolean anonymousaccess = true`)
		err = orca.SetObjSource(pbtFile, pblFile, objName, []byte(src))
		if err != nil {
			return fmt.Errorf("FixHttpClient failed for %s: %v", objName, err)
		}
		return nil
	}
	return step2()
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

	// if lower(ls_exe) = "pb115.exe" or lower(ls_exe) = "pb170.exe" then
	regex := regexp.MustCompile(`(?mi)[ \t]*if[ ]+(lower\([ ]*ls_exe[ ]*\)[ ]*=[ ]*"pb[0-9]{3}\.exe".*?)then[ ]*`)

	matches := regex.FindAllStringSubmatch(src, -1)
	if len(matches) == 0 {
		warnFunc(fmt.Sprintf("skipping %s migration as regex found no match", objName))
		return nil
	}
	if len(matches) != 1 {
		return fmt.Errorf("FixLifProcess failed: exe string is not present in project %s", libFolder)
	}
	if strings.Trim(matches[0][1], " \t") == `lower(ls_exe) = "pb170.exe" or lower(ls_exe) = "pb220.exe" or lower(ls_exe) = "pb250.exe"` {
		warnFunc(fmt.Sprintf("skipping %s migration, already migrated", objName))
		return nil
	}
	if strings.Trim(matches[0][1], " \t") != `lower(ls_exe) = "pb115.exe" or lower(ls_exe) = "pb170.exe"` {
		warnFunc(fmt.Sprintf("  %s in folder %s doesnt contain the expected content (%s)", objName, libFolder, matches[0][1]))
	}

	src = regex.ReplaceAllString(src, `	if lower(ls_exe) = "pb170.exe" or lower(ls_exe) = "pb220.exe" or lower(ls_exe) = "pb250.exe" then`)
	err = orca.SetObjSource(pbtFile, pblFile, objName, []byte(src))
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

	if len(regex.FindAllString(src, -1)) == 0 {
		warnFunc(fmt.Sprintf("skipping %s migration, already migrated", objName))
		return nil
	}

	src = regex.ReplaceAllString(src, `${1}CI${2}`)
	err = orca.SetObjSource(pbtFile, pblFile, objName, []byte(src))
	if err != nil && !ignoreCompileErr {
		return fmt.Errorf("FixLifMetratec failed: %v", err)
	}
	return nil
}

// FixPayrollXmlDecl removes deprecated use of pbdom_processinginstruction
func FixPayrollXmlDecl(libFolder string, targetName string, orca *pborca.Orca, warnFunc func(string)) error {
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
		if len(regex1.FindAllString(src, -1)) == 0 && len(regex2.FindAllString(src, -1)) == 0 {
			warnFunc(fmt.Sprintf("skipping %s migration, already migrated", objName))
			return nil
		}
		src = regex1.ReplaceAllString(src, `ipbdom_document.setxmldeclaration("1.0", "UTF-8", "yes")`)
		src = regex2.ReplaceAllString(src, ``)

		err = orca.SetObjSource(pbtFile, pblFile, objName, []byte(src))
		if err != nil {
			return fmt.Errorf("FixLohXmlDecl failed on %s: %v", objName, err)
		}
	}
	return nil
}

// FixPayrollXmlDecl removes deprecated use of pbdom_processinginstruction
func FixPayrollXmlEncoding(libFolder string, targetName string, orca *pborca.Orca, warnFunc func(string)) error {
	pbtFile := filepath.Join(libFolder, targetName+".pbt")
	pblFile := filepath.Join(libFolder, "loh1.pbl")
	objName := "loh1_u_loh_xml_pbdom"
	src, err := orca.GetObjSource(pblFile, objName)
	if err != nil {
		warnFunc(fmt.Sprintf("skipping %s migration (does not exist in %s)", objName, pblFile))
		return nil
	}
	if strings.Contains(src, `if lpd_obj[1].getobjectclassstring() <> "pbdom_processinginstruction" then`) {
		warnFunc(fmt.Sprintf("skipping %s migration, already migrated", objName))
		return nil
	}
	src = strings.ReplaceAll(src, `apd_doc.GetContent(lpd_obj)`,
		`apd_doc.GetContent(lpd_obj)

if upperbound(lpd_obj) >= 1 then
	if lpd_obj[1].getobjectclassstring() <> "pbdom_processinginstruction" then
		// write XML Declaration manually as first line
		FileWriteEx(ii_filenum, '<?xml version="1.0" encoding="UTF-8"?>')
	end if
end if
`)
	err = orca.SetObjSource(pbtFile, pblFile, objName, []byte(src))
	if err != nil {
		return fmt.Errorf("FixPayrollXmlEncoding failed on %s: %v", objName, err)
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
	objList, err := orca.GetObjList(pblFile)
	if err != nil {
		return fmt.Errorf("AddMirrorObjects failed: %v", err)
	}
	for _, file := range files {
		objSrc, err := mirrorFiles.ReadFile("mirror_objects/" + file.Name())
		if err != nil {
			return fmt.Errorf("AddMirrorObjects failed: %v", err)
		}
		objName := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
		if _, ok := objList[objName]; !ok {
			warnFunc(fmt.Sprintf("skipping import of mirror object %s, it already exists", objName))
			continue
		}
		err = orca.SetObjSource(pbtFile, pblFile, objName, objSrc)
		if err != nil {
			return fmt.Errorf("AddMirrorObjects failed: %v", err)
		}
	}
	return nil
}

// ChangePbdomBuildOptions adds pbdom to the build projects` build list.
// It also removes the old pbdom from the list
func ChangePbdomBuildOptions(projLibName string, projName string, pbtData *orca.Pbt, orca *pborca.Orca, warnFunc func(string)) error {
	pblFile := filepath.Join(pbtData.BasePath, projLibName)
	pbtFile := filepath.Join(pbtData.BasePath, pbtData.AppName+".pbt")

	objName := projName
	src, err := orca.GetObjSource(pblFile, objName+".srj")
	if err != nil {
		warnFunc(fmt.Sprintf("skipping ChangePbdomBuildOption, as the source of the project file %s could not be found in %s(%v)", objName, projLibName, err))
		return nil
	}

	regex := regexp.MustCompile(`(?im)(PBD:pbdom[0-9]+\.pbl,,[01])`)
	if len(regex.FindAllString(src, -1)) == 0 {
		warnFunc("skipping change of pbdom build setting, already migrated")
		return nil
	}

	src = regex.ReplaceAllString(src, `PBD:pbdom.pbl,,1`)
	err = orca.SetObjSource(pbtFile, pblFile, objName, []byte(src))
	if err != nil {
		return fmt.Errorf("ChangePbdomBuildOptions failed: %v", err)
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
			return fmt.Errorf("FixRuntimeFolder failed while getting project source: %v", err)
		}
		src = regexp.MustCompile(`(?mi)^(EXE:.*?)[A-Z]:\\[^,\r\n]+[\r\n]+`).ReplaceAllString(src, "$1.\\pbdk\r\n")
		err = orca.SetObjSource(pbtFile, pblFile, proj.Name, []byte(src))
		if err != nil {
			return fmt.Errorf("FixRuntimeFolder failed while setting project source: %v", err)
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
	err = os.WriteFile(pbtFilePath, pbtData, 0o664)
	if err != nil {
		return fmt.Errorf("FixProjLib failed: %v", err)
	}
	return nil
}

// ReplacePayrollPbwFile replaces the pbwFile (to get rid of other targets)
func ReplacePayrollPbwFile(pbwFilePath string) error {
	return os.WriteFile(pbwFilePath, getPbFile("a3_lohn.pbw"), 0o664)
}
