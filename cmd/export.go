package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/informaticon/dev.win.base.pbmanager/utils"
	pborca "github.com/informaticon/lib.go.base.pborca"
	"github.com/spf13/cobra"
)

// exportCmd represents the export command
var exportCmd = &cobra.Command{
	Use:   "export [options] <pbl path> <object name> [<dst folder>]",
	Short: "Exports an object from a pbl file",
	Long: `If object name is '*', pbmanager exports all objects within the library.
With dst folder, you can specify the path where the object(s) are exportet to.`,
	Args: cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		pblFilePath := args[0]
		objName := args[1]
		if !filepath.IsAbs(pblFilePath) {
			pblFilePath = filepath.Join(basePath, pblFilePath)
		}
		if !utils.FileExists(pblFilePath) || filepath.Ext(pblFilePath) != ".pbl" {
			return fmt.Errorf("file %s does not exist or is not a pbl file", pblFilePath)
		}
		dstFolderPath := filepath.Join(basePath, "src")
		if len(args) > 2 {
			if !filepath.IsAbs(dstFolderPath) {
				dstFolderPath = filepath.Join(basePath, dstFolderPath)
			}
			err := os.MkdirAll(dstFolderPath, os.ModeDir)
			if err != nil {
				return err
			}
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

		if objName == "*" {
			err = exportPbl(Orca, pblFilePath, dstFolderPath)
			if err != nil {
				return err
			}
		} else {
			srcData, err := Orca.GetObjSource(pblFilePath, objName)
			if err != nil {
				return err
			}
			fileName, err := Orca.GetFilenameOfSrc(srcData)
			if err != nil {
				return err
			}
			err = os.WriteFile(fileName, []byte(srcData), 0664)
			if err != nil {
				return err
			}
		}

		fmt.Println("export done")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)
}

func exportPbl(Orca *pborca.Orca, pblFilePath string, dstDir string) error {
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
			err = os.WriteFile(filepath.Join(dstDir, fileName), []byte(srcData), 0664)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
