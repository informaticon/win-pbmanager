package cmd

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/informaticon/dev.win.base.pbmanager/utils"
	pborca "github.com/informaticon/lib.go.base.pborca"
	"github.com/spf13/cobra"
)

// deleteCmd represents the export command
var deleteCmd = &cobra.Command{
	Use:   "delete <pbl path> -n <object name>",
	Short: "Removes an object from a pbl file",
	Long:  `You can define regex patterns or "*" for <object name> `,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pblFilePath := args[0]
		objName, _ := cmd.Flags().GetString("object-name")
		fileType := filepath.Ext(pblFilePath)

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

		if !filepath.IsAbs(pblFilePath) {
			pblFilePath = filepath.Join(basePath, pblFilePath)
		}
		// check if provided pblFilePath exists and is allowed
		if !utils.FileExists(pblFilePath) || fileType != ".pbl" {
			return fmt.Errorf("file %s does not exist or is not a pbl file", pblFilePath)
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

		return deletePbl(Orca, pblFilePath, objRegex)
	},
}

var ignoreMissing bool

func init() {
	rootCmd.AddCommand(deleteCmd)
	deleteCmd.PersistentFlags().StringP("object-name", "n", "*", "name or regex of object to export like 'inf1_u_mail.sru' or 'u_.*'")

	deleteCmd.PersistentFlags().BoolVarP(&ignoreMissing, "ignore-missing", "i", true, "do not fail if object does not exist")
}

func deletePbl(Orca *pborca.Orca, pblFilePath string, objRegex *regexp.Regexp) error {
	if ignoreMissing {
		// outDir = filepath.Join(outDir, filepath.Base(pblFilePath))
	}

	objs, err := Orca.GetObjList(pblFilePath)
	if err != nil {
		return err
	}

	deletedCount := 0
	for _, objArr := range objs {
		for _, obj := range objArr.GetObjArr() {
			objName := obj.GetName() + pborca.GetObjSuffixFromType(obj.GetObjType())
			if objRegex.FindString(objName) == "" {
				continue
			}

			err = Orca.DeleteObj(pblFilePath, objName)
			if err != nil {
				return err
			}
			deletedCount++
			fmt.Printf("deleted %s\n", objName)
		}
	}
	if deletedCount == 0 && !ignoreMissing {
		return fmt.Errorf("no object matching '%s' found in %s", objRegex, pblFilePath)
	}

	return nil
}
