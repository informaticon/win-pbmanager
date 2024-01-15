package cmd

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/informaticon/dev.win.base.pbmanager/migrate"
	"github.com/informaticon/dev.win.base.pbmanager/utils"
	pborca "github.com/informaticon/lib.go.base.pborca"
	"github.com/informaticon/lib.go.base.pborca/orca"
	"github.com/spf13/cobra"
)

// upgradeCmd represents the upgrade command
var upgradeCmd = &cobra.Command{
	Use:   "upgrade <pbt path>",
	Short: "Upgrade (migrate) a PowerBuilder project",
	Long: `Migrate a project from an older PoweBuilder version.
You have to specify the path to the PowerBuilder target (e.g. C:/a3/lib/a3.pbt). The function then applies required patches and performs the migration.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !filepath.IsAbs(args[0]) {
			args[0] = filepath.Join(basePath, args[0])
		}
		if !utils.FileExists(args[0]) {
			return fmt.Errorf("pbt file %s does not exist", args[0])
		}
		if orcaVars.pbVersion != 22 {
			return fmt.Errorf("currently, only PowerBuilder 22 is supported")
		}

		var opts []func(*pborca.Orca)
		opts = append(opts, pborca.WithOrcaTimeout(time.Duration(orcaVars.timeoutSeconds)*time.Second))
		if orcaVars.serverAddr != "" {
			opts = append(opts, pborca.WithOrcaServer(orcaVars.serverAddr, orcaVars.serverApiKey))
		}
		pbtData, err := orca.NewPbtFromFile(args[0])
		if err != nil {
			return err
		}

		err = doUpgrade(pbtData, orcaVars.pbVersion, opts...)
		if err != nil {
			panic(err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(upgradeCmd)
}

func doUpgrade(pbtData *orca.Pbt, pbVersion int, options ...func(*pborca.Orca)) error {
	orca, err := pborca.NewOrca(pbVersion, options...)
	if err != nil {
		return err
	}
	defer orca.Close()

	var libs3rd migrate.Libs3rd

	err = libs3rd.AddMissingLibs(pbtData)
	if err != nil {
		return err
	}
	defer libs3rd.CleanupLibs()

	if pbtData.AppName == "a3" || pbtData.AppName == "loh" {
		err = migrateStepB(pbtData, orca)
		if err != nil {
			return err
		}
		fmt.Println("Step B done")
	} else {
		fmt.Println("Skipping Step B (not an a3/lohn project) ")
	}

	err = migrateStepC(pbtData, orca)
	if err != nil {
		return err
	}
	fmt.Println("Step C done")

	/*err = orca.FullBuildTarget(filepath.Join(pbtData.BasePath, pbtData.AppName+".pbt"))
	if err != nil {
		return err
	}*/

	fmt.Println("Full Build done")
	return nil
}

func migrateStepC(pbtData *orca.Pbt, orca *pborca.Orca) (err error) {
	pbtFilePath := filepath.Join(pbtData.BasePath, pbtData.AppName+".pbt")

	log, err := orca.MigrateTarget(pbtFilePath)
	if err != nil {
		fmt.Println(log)
		return
	}
	err = migrate.FixRuntimeFolder(pbtData, orca, printWarn)
	if err != nil {
		return
	}

	return nil
}

func migrateStepB(pbtData *orca.Pbt, orca *pborca.Orca) (err error) {
	for i, proj := range pbtData.Projects {
		if proj.Name == "a3" && proj.PblFile == "inf2.pbl" {
			_, err := orca.GetObjSource(filepath.Join(pbtData.BasePath, proj.PblFile), "a3.srj")
			if err == nil {
				continue
			}
			err = migrate.FixProjLib(filepath.Join(pbtData.BasePath, pbtData.AppName+".pbt"), proj.Name, "inf2.pbl", "inf1.pbl")
			if err != nil {
				return err
			}
			pbtData.Projects[i].PblFile = "inf1.pbl"

		}
	}
	err = migrate.AddMirrorObjects(pbtData.BasePath, pbtData.AppName, orca, printWarn)
	if err != nil {
		return
	}
	if pbtData.AppName == "a3" {
		// lohn has no registry object
		err = migrate.FixRegistry(pbtData.BasePath, pbtData.AppName, orca, printWarn)
		if err != nil {
			return
		}
	}
	err = migrate.FixLibInterface(pbtData.BasePath, pbtData.AppName, orca, printWarn)
	if err != nil {
		return
	}

	err = migrate.RemoveFiles(pbtData.BasePath, printWarn)
	if err != nil {
		return
	}

	err = migrate.InsertNewPbdk(pbtData.BasePath)
	if err != nil {
		return
	}

	err = migrate.InsertNewPbdom(pbtData.BasePath, pbtData.AppName)
	if err != nil {
		return
	}

	err = migrate.FixPbInit(pbtData.BasePath, printWarn)
	if err != nil {
		return
	}

	if pbtData.AppName == "loh" {
		err = migrate.ReplacePayrollPbwFile(filepath.Join(pbtData.BasePath, "a3_lohn.pbw"))
		if err != nil {
			return
		}
	}

	return
}

func printWarn(message string) {
	fmt.Println(message)
}
