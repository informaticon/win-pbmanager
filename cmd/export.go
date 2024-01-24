package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/informaticon/dev.win.base.pbmanager/utils"
	pborca "github.com/informaticon/lib.go.base.pborca"
	"github.com/informaticon/lib.go.base.pborca/orca"
	"github.com/spf13/cobra"
)

// exportCmd represents the export command
var exportCmd = &cobra.Command{
	Use:   "export [options] <pbl/pbt path> --object-name --output-dir",
	Short: "Exports objects from a pbl/pbt file",
	Long: `If object name is '*', pbmanager exports all objects within the library.
With --output-dir, you can specify the path where the object(s) are exportet to.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		objFilePath := args[0]
		objName, _ := cmd.Flags().GetString("object-name")
		outputDirectory, _ := cmd.Flags().GetString("output-dir")
		fileType := filepath.Ext(objFilePath)

		if !filepath.IsAbs(objFilePath) {
			objFilePath = filepath.Join(basePath, objFilePath)
		}
		//check if provided objFilePath exists and is allowed
		if !utils.FileExists(objFilePath) || (fileType != ".pbl" && fileType != ".pbt") {
			return fmt.Errorf("file %s does not exist or is not a pbl/pbt file", objFilePath)
		}

		//if no output directory is provided, store the export along side the objFilePath in the src folder
		if outputDirectory == "" {
			outputDirectory = filepath.Join(filepath.Dir(objFilePath), "src")
		}
		err := os.MkdirAll(outputDirectory, os.ModeDir)
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

		if fileType == ".pbt" {
			err = exportPbt(Orca, objFilePath, outputDirectory)
			if err != nil {
				return err
			}
		} else {
			if objName == "*" {
				err = exportPbl(Orca, objFilePath, outputDirectory)
				if err != nil {
					return err
				}

			} else {
				srcData, err := Orca.GetObjSource(objFilePath, objName)
				if err != nil {
					return err
				}
				fileName, err := Orca.GetFilenameOfSrc(srcData)
				if err != nil {
					return err
				}
				err = os.WriteFile(filepath.Join(outputDirectory, fileName), []byte(srcData), 0664)
				if err != nil {
					return err
				}
			}
		}

		fmt.Println("export finished")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)
	exportCmd.PersistentFlags().String("object-name", "*", "inf1_u_application")
	//exportCmd.PersistentFlags().Lookup("object-name").NoOptDefVal = "*"
	exportCmd.PersistentFlags().String("output-dir", "", "path to output directory")
}

func exportPbl(Orca *pborca.Orca, pblFilePath string, outputDirectory string) error {
	objs, err := Orca.GetObjList(pblFilePath)
	if err != nil {
		return err
	}
	for _, objArr := range objs {
		for _, obj := range objArr.GetObjArr() {
			srcData, err := Orca.GetObjSource(pblFilePath, obj.GetName()+pborca.GetObjSuffixFromType(obj.GetObjType()))
			if err != nil {
				return err
			}
			fileName, err := Orca.GetFilenameOfSrc(srcData)
			if err != nil {
				return err
			}
			err = os.WriteFile(filepath.Join(outputDirectory, fileName), []byte(srcData), 0664)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func exportPbt(Orca *pborca.Orca, pbtFilePath string, outputDirectory string) error {
	pbt, err := orca.NewPbtFromFile(pbtFilePath)
	if err != nil {
		return err
	}
	for _, lib := range pbt.LibList {
		libName := filepath.Base(lib)

		pblOutputDir := filepath.Join(outputDirectory, libName)
		err := os.MkdirAll(pblOutputDir, os.ModeDir)
		if err != nil {
			return err
		}
		fmt.Print("Exporting library ", libName)
		err = exportPbl(Orca, lib, pblOutputDir)
		if err != nil {
			return err
		}
		fmt.Println(" done.")
	}

	return nil
}
