/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/informaticon/dev.win.base.pbmanager/utils"
	pborca "github.com/informaticon/lib.go.base.pborca"
	"github.com/spf13/cobra"
)

var pbtFilePath string

// importCmd represents the import command
var importCmd = &cobra.Command{
	Use:   "import [options] <pbl path> [<src file path>|<src folder path>]",
	Short: "Imports one or multiple source files into a libtaty",
	Long: `To import a source file, pbmanager needs to know the PowerBuilder target (pbt-file).
You can set the path to the target or let pbmanager try to find the target.
Examples:
	- pbmanager import -b C:/a3/lib -t liq.pbt tst1.pbl src/w_main.srw
	- pbmanager import -b C:/a3/lib tst1.pbl src/
	- pbmanager import C:/a3/lib/liq.pbt C:/a3/lib/tst1.pbl C:/tst1_u_tst_main.sru`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		pblFilePath := args[0]
		srcPath := args[1]
		if !filepath.IsAbs(pblFilePath) {
			pblFilePath = filepath.Join(basePath, pblFilePath)
		}
		if !utils.FileExists(pblFilePath) || filepath.Ext(pblFilePath) != ".pbl" {
			return fmt.Errorf("file %s does not exist or is not a pbl file", pblFilePath)
		}
		if !filepath.IsAbs(srcPath) {
			srcPath = filepath.Join(basePath, srcPath)
		}
		if !utils.FileExists(srcPath) {
			return fmt.Errorf("path %s does not exist", srcPath)
		}
		pbtFilePath, err = findPbtFilePath(basePath, pbtFilePath)
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

		if isFile(srcPath) {
			srcData, err := os.ReadFile(srcPath)
			if err != nil {
				return err
			}
			err = Orca.SetObjSource(pbtFilePath, pblFilePath, filepath.Base(srcPath), string(srcData))
			if err != nil {
				return err
			}
		} else {
			err = filepath.WalkDir(srcPath, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				srcData, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				err = Orca.SetObjSource(pbtFilePath, pblFilePath, filepath.Base(srcPath), string(srcData))
				return err
			})
			if err != nil {
				return err
			}
		}

		fmt.Println("import finished")
		return nil
	},
}

func init() {
	importCmd.Flags().StringVarP(&pbtFilePath, "target", "t", "", "Target file to use (e.g. C:/a3/lib/a3.pbt). If omitted, pbmanagers tries to find the appropriate taget automatically.")
	rootCmd.AddCommand(importCmd)
}
