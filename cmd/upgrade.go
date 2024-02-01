package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/informaticon/dev.win.base.pbmanager/migrate"
	"github.com/informaticon/dev.win.base.pbmanager/utils"
	pborca "github.com/informaticon/lib.go.base.pborca"
	"github.com/informaticon/lib.go.base.pborca/orca"
	"github.com/informaticon/lib.go.base.pborca/pbc"
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
			fmt.Println(err)
			os.Exit(2)
		}
		return nil
	},
}

func init() {
	utils.GetRessource("https://choco.informaticon.com/endpoints/axp/content/lib.bin.base.pbdk@22.2.0-3289.zip")
	rootCmd.AddCommand(upgradeCmd)
}

// buildWithPbc uses pbc.exe to build the project.
// This gives better error messages than orca.
func buildWithPbc(pbtPath string) string {
	compiler, err := pbc.NewPBCompiler(
		pbtPath, pbc.Pb22, pbc.WithCompileMethod(pbc.CompileMethodCompile),
	)
	if err != nil {
		return err.Error()
	}
	log, err := compiler.Run()
	if err != nil {
		return fmt.Sprintf("Build with pbc220.exe failed, compiler log:\n%s", log)
	}
	return fmt.Sprintf("Build with pbc220.exe was successfull, compiler log:\n%s", log)
}
func doUpgrade(pbtData *orca.Pbt, pbVersion int, options ...func(*pborca.Orca)) error {
	orca, err := pborca.NewOrca(pbVersion, options...)
	if err != nil {
		return err
	}
	defer orca.Close()

	err = migrate.InsertExfInPbt(pbtData, orca)
	if err != nil {
		return err
	}

	var libs3rd migrate.Libs3rd

	err = libs3rd.AddMissingLibs(pbtData)
	if err != nil {
		return err
	}
	defer libs3rd.CleanupLibs()

	err = preMigrateFromPb115(pbtData, orca, printWarn)
	if err != nil {
		fmt.Println(buildWithPbc(pbtData.GetPath()))
		return err
	}

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
		fmt.Println(buildWithPbc(pbtData.GetPath()))
		return err
	}
	fmt.Println("Step C done")

	dat, err := orca.FullBuildTarget(filepath.Join(pbtData.BasePath, pbtData.AppName+".pbt"))
	if err != nil {
		return fmt.Errorf("%s\n%v", strings.Join(dat, "\n"), err)
	}

	fmt.Println("Full Build done")
	return nil
}

func migrateStepC(pbtData *orca.Pbt, orca *pborca.Orca) (err error) {
	pbtFilePath := filepath.Join(pbtData.BasePath, pbtData.AppName+".pbt")

	out, err := orca.MigrateTarget(pbtFilePath)
	if err != nil {
		return fmt.Errorf("migration of %s failed, compiler log\n%s\nORCA Error:%v", pbtFilePath, strings.Join(out, "\n"), err)
	}

	err = migrate.FixRuntimeFolder(pbtData, orca, printWarn)
	if err != nil {
		return
	}
	for _, proj := range pbtData.Projects {
		err = migrate.ChangePbdomBuildOptions(proj.PblFile, proj.Name, pbtData, orca, printWarn)
		if err != nil && slices.Contains([]string{"a3", "loh"}, proj.Name) {
			return
		}
	}

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

	if pbtData.AppName == "a3" {
		// lohn has no registry object
		err = migrate.FixRegistry(pbtData.BasePath, pbtData.AppName, orca, printWarn)
		if err != nil {
			return
		}
	}

	err = migrate.FixLifProcess(pbtData.BasePath, pbtData.AppName, orca, printWarn)
	if err != nil {
		return
	}

	err = migrate.AddMirrorObjects(pbtData.BasePath, pbtData.AppName, orca, printWarn)
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

	if pbtData.AppName == "loh" {
		err = migrate.FixPayrollXmlDecl(pbtData.BasePath, pbtData.AppName, orca, printWarn)
		if err != nil {
			return
		}
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

	if utils.FileExists(filepath.Join(pbtData.BasePath, pbtData.AppName+".exe")) {
		os.Remove(filepath.Join(pbtData.BasePath, pbtData.AppName+".exe"))
	}

	return
}

// preMigrateFromPb115 does some steps which are needed to migrate from PB115
func preMigrateFromPb115(pbtData *orca.Pbt, orca *pborca.Orca, warnFunc func(string)) (err error) {
	pblFile := filepath.Join(pbtData.BasePath, "lif1.pbl")
	pbtFile := filepath.Join(pbtData.BasePath, pbtData.AppName+".pbt")

	objName := "lif1_u_metratec_base"
	src, err := orca.GetObjSource(pblFile, objName)
	if err != nil {
		objName = "inf1_u_metratec_base"
		src, err = orca.GetObjSource(pblFile, objName)
		if err != nil {
			return
		}
	}
	warnFunc("Start PB115 pre migration")

	regex := regexp.MustCompile(`(?im)([ \t])(_INFO|_FATAL|_ERROR|_DEBUG|_WARN)`)
	src = regex.ReplaceAllString(src, `${1}CI${2}`)

	err = migrate.InsertNewPbdom(pbtData.BasePath, pbtData.AppName)
	if err != nil {
		return
	}

	err = orca.SetObjSource(pbtFile, pblFile, objName, src)
	if err != nil {
		fmt.Printf("info: SetObjSource for preMigration of PB115 failed, this can be ignored (%v)\n", err)
	}

	err = migrateStepC(pbtData, orca)
	if err != nil {
		return
	}

	err = orca.SetObjSource(pbtFile, pblFile, objName, src)
	if err != nil {
		return
	}

	fmt.Println("PB115 pre migration finished")
	return nil
}

func printWarn(message string) {
	fmt.Println(message)
}
