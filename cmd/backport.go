package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/informaticon/dev.win.base.pbmanager/internal/backport"
	"github.com/informaticon/dev.win.base.pbmanager/internal/importer"
	pborca "github.com/informaticon/lib.go.base.pborca"
	"github.com/spf13/cobra"
)

// backportCmd represents the conversion back from solution to workspace
var backportCmd = &cobra.Command{
	Use:   "backport <some.pbsln | some.pbproj> [options] ",
	Short: "performs the conversion back from solution or project to workspace or target",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := os.Stat(args[0]); err != nil {
			return fmt.Errorf("no .pbsln or .pbproj file was provided: %v", err)
		}
		absoluteSolOrProjPath, err := filepath.Abs(args[0])
		if err != nil {
			return err
		}
		if filepath.Ext(absoluteSolOrProjPath) != ".pbproj" && filepath.Ext(absoluteSolOrProjPath) != ".pbsln" {
			return fmt.Errorf("no .pbsln or .pbproj file was provided: %v", err)
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
		var importerOpts []func(*importer.MultiImport)
		importerOpts = append(importerOpts, importer.WithOrcaOpts(opts))
		importerOpts = append(importerOpts, importer.WithNumberWorkers(numberWorkers))
		importerOpts = append(importerOpts, importer.WithMinIterations(minIterations))
		if filepath.Ext(absoluteSolOrProjPath) == ".pbsln" {
			return backport.ConvertSolutionToWorkspace(absoluteSolOrProjPath, importerOpts)
		} else {
			return backport.ConvertProjectToTarget(absoluteSolOrProjPath, importerOpts)
		}
	},
}

var numberWorkers int
var minIterations int

func init() {
	rootCmd.AddCommand(backportCmd)
	backportCmd.Flags().IntVarP(&numberWorkers, "workers", "w", 1,
		"number of parallel processed PBL sources / imports.")
	backportCmd.Flags().IntVar(&minIterations, "min-iter", 15, "number of iterations "+
		"through all PBL sources when errors occur.")
}
