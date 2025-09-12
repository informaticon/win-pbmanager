package cmd

import (
	"fmt"
	"time"

	"github.com/informaticon/dev.win.base.pbmanager/internal/backport"
	pborca "github.com/informaticon/lib.go.base.pborca"
	"github.com/spf13/cobra"
)

// backportCmd represents the conversion back from solution to workspace
var backportCmd = &cobra.Command{
	Use:   "backport <some.pbproj> [options] ",
	Short: "performs the conversion back from PB project to target",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var absoluteProjPath string

		if len(args) == 0 {
			absoluteProjPath = ""
		} else {
			absoluteProjPath = args[0]
		}
		absoluteProjPath, err := findPbProjFilePath(basePath, absoluteProjPath)
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
		defer Orca.Close()

		return backport.ConvertProjectToTarget(Orca, absoluteProjPath)
	},
}

var minIterations int

func init() {
	rootCmd.AddCommand(backportCmd)
	backportCmd.Flags().IntVar(&minIterations, "min-iter", 15, "number of iterations through all PBL sources when errors occur.")
}
