package cmd

import (
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/informaticon/dev.win.base.pbmanager/utils"
	pborca "github.com/informaticon/lib.go.base.pborca"
	"github.com/spf13/cobra"
)

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build [options] <pbt path>",
	Short: "Builds a PowerBuilder target",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pbtFilePath := args[0]
		if !filepath.IsAbs(pbtFilePath) {
			pbtFilePath = filepath.Join(basePath, pbtFilePath)
		}
		if !utils.FileExists(pbtFilePath) || filepath.Ext(pbtFilePath) != ".pbt" {
			return fmt.Errorf("file %s does not exist or is not a pbt file", pbtFilePath)
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
		logs, err := Orca.FullBuildTarget(pbtFilePath)
		if len(logs) > 0 {
			log.Printf("Compiler Log:\n%v\n", logs)
		}
		if err != nil {
			return err
		}
		fmt.Println("Build done")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)
}
