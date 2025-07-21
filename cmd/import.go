package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/informaticon/dev.win.base.pbmanager/internal/importer"
	"github.com/informaticon/dev.win.base.pbmanager/utils"
	pborca "github.com/informaticon/lib.go.base.pborca"
	"github.com/informaticon/lib.go.base.pborca/orca"
	"github.com/spf13/cobra"
)

var (
	pbtFilePath string
	pblList     []string
)

// importCmd represents the import command
var importCmd = &cobra.Command{
	Use:   "import [options] <pbl path> [<src file paths>...|<src folder paths>...]",
	Short: "Imports one or multiple source files into a library",
	Long: `To import a source file, pbmanager needs to know the PowerBuilder target (pbt-file).
You can set the path to the target or let pbmanager try to find the target.
Usually, you have to declare the pbl file into which you want to import the source,
but you can also just specify a pbt and a list of pbl names (-p parameter). In this case,
pbmanager tries to import all the pbl multiple times until there is no compilation error.
Examples:
	- pbmanager import -b C:/a3/lib -t liq.pbt tst1.pbl src/w_main.srw
	- pbmanager import -b C:/a3/lib tst1.pbl src/
	- pbmanager import C:/a3/lib/liq.pbt C:/a3/lib/tst1.pbl C:/tst1_u_tst_main.sru
	- pbmanager import my.pbt -p tst1,exf1,str1 . C:/additional/src_folder C:/third/src`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		var pblSrcFilePath string
		var srcPaths []string

		if len(pblList) == 0 {
			// pbl import mode
			if len(args) == 1 {
				return fmt.Errorf("at least 2 positional arguments needed, but got only 1")
			}
			pblSrcFilePath = args[0]
			srcPaths = args[1:]
			if !filepath.IsAbs(pblSrcFilePath) {
				pblSrcFilePath = filepath.Join(basePath, pblSrcFilePath)
			}
			if !utils.FileExists(pblSrcFilePath) || filepath.Ext(pblSrcFilePath) != ".pbl" {
				return fmt.Errorf("file %s does not exist or is not a pbl file", pblSrcFilePath)
			}
		} else {
			// pbt import mode
			srcPaths = args[0:]
		}
		for i := range srcPaths {
			if !filepath.IsAbs(srcPaths[i]) {
				srcPaths[i] = filepath.Join(basePath, srcPaths[i])
			}
			if !utils.FileExists(srcPaths[i]) {
				return fmt.Errorf("path %s does not exist", srcPaths[i])
			}
		}

		pbtFilePath, err = findPbtFilePath(basePath, pbtFilePath)
		if err != nil {
			return err
		}

		if orcaVars.pbVersion != 22 {
			return fmt.Errorf("currently, only PowerBuilder 22 is supported")
		}
		var opts []func(*pborca.Orca)
		if orcaVars.pbRuntimeFolder != "" {
			opts = append(opts, pborca.WithOrcaRuntime(orcaVars.pbRuntimeFolder))
		}
		opts = append(opts, pborca.WithOrcaTimeout(time.Duration(orcaVars.timeoutSeconds)*time.Second))
		if orcaVars.serverAddr != "" {
			opts = append(opts, pborca.WithOrcaServer(orcaVars.serverAddr, orcaVars.serverApiKey))
		}
		Orca, err := pborca.NewOrca(orcaVars.pbVersion, opts...)
		if err != nil {
			return err
		}

		if isFile(srcPaths[0]) {
			// pbl import mode - single file
			for _, srcPath := range srcPaths {
				srcData, err := os.ReadFile(srcPath)
				if err != nil {
					return err
				}
				err = Orca.SetObjSource(pbtFilePath, pblSrcFilePath, filepath.Base(srcPath), srcData)
				if err != nil {
					return fmt.Errorf("could not import %s: %w", filepath.Base(srcPath), err)
				}
			}
		} else if len(pblList) == 0 {
			// pbl import mode - folder
			errs := make(map[string]error)
			for _, srcPath := range srcPaths {
				err = filepath.WalkDir(srcPath, func(path string, d fs.DirEntry, err error) error {
					if err != nil {
						return err
					}
					if d.IsDir() {
						return nil
					}
					srcData, err := utils.ReadPbSource(path)
					if err != nil {
						return err
					}
					objName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(filepath.Base(path)))
					err = Orca.SetObjSource(pbtFilePath, pblSrcFilePath, filepath.Base(objName), srcData)
					if err != nil {
						errs[objName] = err
					}
					return nil
				})
				if err != nil {
					return err
				}
			}
			if len(errs) > 0 {
				fmt.Printf("compilation errors occured: %v\n", errs)
			}
		} else /* len(pblList) > 0 */ {
			// pbt import modde - multiple pbl
			pbtData, err := orca.NewPbtFromFile(pbtFilePath)
			if err != nil {
				return err
			}
			var pblFilePaths, pblSrcFilePaths []string

			findPbl := func(pbl string) string {
				for _, srcPath := range srcPaths {
					if utils.FileExists(filepath.Join(srcPath, pbl+".pbl")) {
						return filepath.Join(srcPath, pbl+".pbl")
					}
					if utils.FileExists(filepath.Join(srcPath, pbl+".pbl.src")) {
						return filepath.Join(srcPath, pbl+".pbl.src")
					}
				}
				return ""
			}
			for _, pbl := range pblList {
				pblSrcFilePath := findPbl(pbl)
				if pblSrcFilePath == "" {
					return fmt.Errorf("could not find source folder for %s", pbl)
				}
				for _, pblPath := range pbtData.LibList {
					if filepath.Base(pblPath) == pbl+".pbl" {
						pblSrcFilePaths = append(pblSrcFilePaths, pblSrcFilePath)
						pblFilePaths = append(pblFilePaths, pblPath)
					}
				}
			}
			m := importer.NewMultiImport(pbtFilePath, pblFilePaths, pblSrcFilePaths,
				importer.WithNumberWorkers(1),
				importer.WithOrcaOpts(opts))
			err = m.Import()
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
	importCmd.Flags().StringSliceVarP(&pblList, "pbl-list", "p", pblList, "List of pbl to import (try multiple times until there is no compilation error.")
	rootCmd.AddCommand(importCmd)
}
