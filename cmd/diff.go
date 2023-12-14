/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	pborca "github.com/informaticon/lib.go.base.pborca"
	"github.com/spf13/cobra"
)

var mergeTool string
var nameBase string
var nameMine string
var nameTheirs string

// https://tortoisesvn.net/docs/release/TortoiseSVN_en/tsvn-dug-settings.html

// diffCmd represents the diff command
var diffCmd = &cobra.Command{
	Use:   "diff <pbl base> <pbl mine> [<pbl theirs>] [<pbl merged>]",
	Short: "Compares two or three pbl files",
	Long:  `Export the source of two pbl files and opens WinMerge to show the differences.`,
	Args:  cobra.RangeArgs(2, 4),
	RunE: func(cmd *cobra.Command, args []string) error {
		err := run(args)
		if err != nil {
			fmt.Println(err)
		}
		return err
	},
}

func run(args []string) error {
	tempDir := filepath.Join(os.TempDir(), "pbdiff", time.Now().Format("20170907_170606"))
	os.MkdirAll(tempDir, 0664)
	defer os.RemoveAll(tempDir)

	pblFilePathBase := filepath.Clean(args[0])
	pblFilePathMine := filepath.Clean(args[1])
	pblFilePathTheirs := ""
	pblFilePathMerged := ""
	mergeTool = filepath.Clean(mergeTool)
	if len(args) >= 3 {
		pblFilePathTheirs = filepath.Clean(args[2])
		if !filepath.IsAbs(pblFilePathTheirs) {
			pblFilePathTheirs = filepath.Join(basePath, pblFilePathTheirs)
		}
		if !isPblFile(pblFilePathTheirs) {
			return fmt.Errorf("file %s does not exist or is not a pbl file", pblFilePathTheirs)
		}
	}
	if len(args) >= 4 {
		pblFilePathMerged = filepath.Clean(args[3])
		if !filepath.IsAbs(pblFilePathMerged) {
			pblFilePathMerged = filepath.Join(basePath, pblFilePathMerged)
		}
		if !isPblFile(pblFilePathMerged) {
			return fmt.Errorf("file %s does not exist or is not a pbl file", pblFilePathMerged)
		}
	}
	if !filepath.IsAbs(pblFilePathBase) {
		pblFilePathBase = filepath.Join(basePath, pblFilePathBase)
	}
	if !isPblFile(pblFilePathBase) {
		return fmt.Errorf("file %s does not exist or is not a pbl file", pblFilePathBase)
	}
	if !filepath.IsAbs(pblFilePathMine) {
		pblFilePathMine = filepath.Join(basePath, pblFilePathMine)
	}
	if !isPblFile(pblFilePathMine) {
		return fmt.Errorf("file %s does not exist or is not a pbl file", pblFilePathMine)
	}

	if orcaVars.pbVersion != 22 {
		return fmt.Errorf("currently, only PowerBuilder 22 is supported")
	}
	var opts []func(*pborca.Orca)
	opts = append(opts, pborca.WithOrcaTimeout(time.Duration(orcaVars.timeoutSeconds)*time.Second))
	opts = append(opts, pborca.WithMessageCallback(func(level uint32, msg string) {
		log.Printf("%d: %s\n", level, msg)
	}))
	if orcaVars.serverAddr != "" {
		opts = append(opts, pborca.WithOrcaServer(orcaVars.serverAddr, orcaVars.serverApiKey))
	}
	Orca, err := pborca.NewOrca(orcaVars.pbVersion, opts...)
	if err != nil {
		return err
	}
	defer Orca.Close()

	pblSrcPathBase := filepath.Join(tempDir, fmt.Sprintf("%s (%s)", filepath.Base(pblFilePathBase), getPblFileDescr(pblFilePathBase)))
	os.MkdirAll(pblSrcPathBase, 0664)
	err = exportPbl(Orca, pblFilePathBase, pblSrcPathBase)
	if err != nil {
		return err
	}
	pblSrcPathMine := filepath.Join(tempDir, fmt.Sprintf("%s (%s)", filepath.Base(pblFilePathMine), getPblFileDescr(pblFilePathMine)))
	os.MkdirAll(pblSrcPathMine, 0664)
	err = exportPbl(Orca, pblFilePathMine, pblSrcPathMine)
	if err != nil {
		return err
	}

	if pblFilePathTheirs == "" {
		command := exec.Command(mergeTool, "/r", "/x", "/u", "/ignoreblanklines", pblSrcPathMine, pblSrcPathBase, "/dl", nameMine, "/dr", nameBase)

		out, err := command.CombinedOutput()
		if err != nil {
			return err
		}
		if len(out) > 0 {
			fmt.Println(out)
		}
		return nil
	}

	pblSrcPathTheirs := filepath.Join(tempDir, fmt.Sprintf("%s (%s)", filepath.Base(pblFilePathTheirs), getPblFileDescr(pblFilePathTheirs)))
	os.MkdirAll(pblSrcPathTheirs, 0664)
	err = exportPbl(Orca, pblFilePathTheirs, pblSrcPathTheirs)
	if err != nil {
		return err
	}
	command := exec.Command(mergeTool, "/r", "/x", "/u", "/ignoreblanklines", "/dl", nameMine, "/dm", nameBase, "/dr", nameTheirs, pblSrcPathMine, pblSrcPathBase, pblSrcPathTheirs)

	out, err := command.CombinedOutput()
	if err != nil {
		return err
	}
	if len(out) > 0 {
		fmt.Println(out)
	}

	fmt.Println("Do you want to read back in the merge result? (The base file from the middle will be imported). [y/N]")
	reader := bufio.NewReader(os.Stdin)
	str, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	if !regexp.MustCompile(`(?i)(y|j)[\r\n]+`).MatchString(str) {
		return nil
	}
	pbtFilePath, err := findPbtFilePath(filepath.Dir(pblFilePathBase), "")
	if err != nil {
		return err
	}
	fmt.Printf("Starting import into %s with target %s\n", pblFilePathBase, pbtFilePath)
	srcFiles, err := os.ReadDir(pblSrcPathBase)
	if err != nil {
		return err
	}
	var errs []error
	for _, srcFile := range srcFiles {
		objName := filepath.Base(srcFile.Name())
		objName = strings.TrimSuffix(objName, filepath.Ext(objName))

		srcData, err := os.ReadFile(filepath.Join(pblSrcPathBase, srcFile.Name()))
		if err != nil {
			return err
		}

		err = Orca.SetObjSource(pbtFilePath, pblFilePathMine, objName, string(srcData))
		if err == nil {
			fmt.Printf("Successfully imported %s\n", objName)
		} else {
			fmt.Printf("Import of %s failed: %v\n", objName, err)
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("import finished with %d errors", len(errs))
	}
	fmt.Println("Import finished\n")
	return nil
}
func init() {
	diffCmd.Flags().StringVar(&mergeTool, "winmerge-path", "C:/Program Files/WinMerge/WinMergeU.exe", "Path to WinMergeU.exe.")
	diffCmd.Flags().StringVar(&nameMine, "mine-name", "Mine", "Description in WinMerge for the mine file")
	diffCmd.Flags().StringVar(&nameBase, "base-name", "Base", "Description in WinMerge for the base file")
	diffCmd.Flags().StringVar(&nameTheirs, "theirs-name", "Theirs", "Description in WinMerge for the theirs file")
	rootCmd.AddCommand(diffCmd)
}
func getPblFileDescr(pblFilePath string) string {
	pblFilePath = filepath.Dir(pblFilePath)
	pblFilePath = strings.ReplaceAll(pblFilePath, ":\\", "_")
	pblFilePath = strings.ReplaceAll(pblFilePath, "\\", "_")
	pblFilePath = strings.ReplaceAll(pblFilePath, "/", "_")
	pblFilePath = strings.ReplaceAll(pblFilePath, ":", "_")
	return pblFilePath
}
