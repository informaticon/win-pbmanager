package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/informaticon/dev.win.base.pbmanager/utils"
	pborca "github.com/informaticon/lib.go.base.pborca"
	"github.com/informaticon/lib.go.base.pborca/orca"
	"github.com/spf13/cobra"
)

// exportCmd represents the export command
var exportCmd = &cobra.Command{
	Use:   "export <pbl/pbt path>",
	Short: "Exports objects from a pbl/pbt file",
	Long: `If --object-name is omitted, pbmanager exports all objects within the library.
With --output-dir, you can specify the path where the object(s) are exportet to.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pbxFilePath := args[0]
		objName, _ := cmd.Flags().GetString("object-name")
		exportOutputDir, _ := cmd.Flags().GetString("output-dir")
		fileType := filepath.Ext(pbxFilePath)

		// check/create obj regex
		if objName == "*" || objName == "" {
			objName = "^.*$"
		}
		if !strings.HasSuffix(objName, "$") {
			objName += "$"
		}
		if !strings.HasPrefix(objName, "^") {
			objName = "^" + objName
		}
		objRegex, err := regexp.Compile(objName)
		if err != nil {
			return err
		}

		//check if user passed the create-subdir flag as it has no effect when exporting .pbt files
		if cmd.Flags().Lookup("create-subdir").Changed && fileType == ".pbt" {
			fmt.Println("--create-subdir has no effect when exporting .pbt")
		}

		if !filepath.IsAbs(pbxFilePath) {
			pbxFilePath = filepath.Join(basePath, pbxFilePath)
		}
		//check if provided objFilePath exists and is allowed
		if !utils.FileExists(pbxFilePath) || (fileType != ".pbl" && fileType != ".pbt") {
			return fmt.Errorf("file %s does not exist or is not a pbl/pbt file", pbxFilePath)
		}

		//if no output directory is provided, store the export along side the objFilePath in the src folder
		if exportOutputDir == "" {
			exportOutputDir = filepath.Join(filepath.Dir(pbxFilePath), "src")
		}
		err = os.MkdirAll(exportOutputDir, os.ModeDir)
		if err != nil {
			return err
		}

		if orcaVars.pbVersion != 22 {
			return fmt.Errorf("currently, only PowerBuilder 22 is supported")
		}
		var opts []func(*pborca.Orca)
		opts = append(opts, pborca.WithOrcaTimeout(time.Duration(orcaVars.timeoutSeconds)*time.Second))
		if orcaVars.serverAddr != "" {
			opts = append(opts, pborca.WithOrcaServer(orcaVars.serverAddr, orcaVars.serverApiKey))
		}
		Orca, err := pborca.NewOrca(orcaVars.pbVersion, opts...)
		if err != nil {
			return err
		}
		defer Orca.Close()

		if fileType == ".pbt" {
			err = exportPbt(Orca, pbxFilePath, objRegex, exportOutputDir)
			if err != nil {
				return err
			}
		} else {
			err = exportPbl(Orca, pbxFilePath, objRegex, exportOutputDir)
			if err != nil {
				return err
			}
		}

		fmt.Println("export finished")
		return nil
	},
}

var exportCreateSupdir bool

func init() {
	rootCmd.AddCommand(exportCmd)
	exportCmd.PersistentFlags().StringP("object-name", "n", "*", "name or regex of object to export like 'inf1_u_mail.sru' or 'u_.*'")
	exportCmd.PersistentFlags().StringP("output-dir", "o", "", "path to output directory (default is <pbl/pbt path>/src")
	exportCmd.PersistentFlags().BoolVarP(&exportCreateSupdir, "create-subdir", "s", true, "create a subfolder with the library name to export the source file(s) into")
}

func exportPbl(Orca *pborca.Orca, pblFilePath string, objRegex *regexp.Regexp, outputDirectory string) error {
	if exportCreateSupdir {
		outputDirectory = filepath.Join(outputDirectory, filepath.Base(pblFilePath))
	}

	objs, err := Orca.GetObjList(pblFilePath)
	if err != nil {
		return err
	}

	var dirCreated = false
	for _, objArr := range objs {
		for _, obj := range objArr.GetObjArr() {
			objName := obj.GetName() + pborca.GetObjSuffixFromType(obj.GetObjType())
			if objRegex.FindString(objName) == "" {
				continue
			}
			srcData, err := Orca.GetObjSource(pblFilePath, objName)
			if err != nil {
				return err
			}
			fileName, err := Orca.GetFilenameOfSrc(srcData)
			if err != nil {
				return err
			}
			if !dirCreated {
				fmt.Printf("Exporting library %s\n", filepath.Base(filepath.Base(pblFilePath)))
				err := os.MkdirAll(outputDirectory, os.ModeDir)
				if err != nil {
					return err
				}
				dirCreated = true
			}
			err = os.WriteFile(filepath.Join(outputDirectory, fileName), []byte(srcData), 0664)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func exportPbt(Orca *pborca.Orca, pbtFilePath string, objRegex *regexp.Regexp, outputDirectory string) error {
	pbt, err := orca.NewPbtFromFile(pbtFilePath)
	if err != nil {
		return err
	}
	for _, lib := range pbt.LibList {
		if !utils.FileExists(lib) {
			fmt.Printf("Library %s does not exist, skipping.....\n", lib)
			continue
		}
		err = exportPbl(Orca, lib, objRegex, outputDirectory)
		if err != nil {
			return err
		}
	}

	return nil
}

func exportPbtWg(Orca *pborca.Orca, pbtFilePath string, objRegex *regexp.Regexp, outputDirectory string, wg *sync.WaitGroup) error {
	defer wg.Done()
	err := exportPbt(Orca, pbtFilePath, objRegex, outputDirectory)
	//wg.Wait()
	return err
}

func exportPblWg(Orca *pborca.Orca, pblFilePath string, objRegex *regexp.Regexp, outputDirectory string, wg *sync.WaitGroup) error {
	defer wg.Done()
	err := exportPbl(Orca, pblFilePath, objRegex, outputDirectory)
	//wg.Wait()
	return err
}
