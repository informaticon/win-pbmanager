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
	"sync"
	"time"

	pborca "github.com/informaticon/lib.go.base.pborca"
	"github.com/informaticon/lib.go.base.pborca/orca"
	"github.com/spf13/cobra"
)

var mergeTool string
var nameBase string
var nameMine string
var nameTheirs string

type exportJob struct {
	libraryPath     string
	destinationPath string
}

// https://tortoisesvn.net/docs/release/TortoiseSVN_en/tsvn-dug-settings.html

// diffCmd represents the diff command
var diffCmd = &cobra.Command{
	Use:   "diff <pbl base> <pbl mine> [<pbl theirs>] [<pbl merged>]",
	Short: "Compares two or three pbl files",
	Long:  `Export the source of two pbl files and opens WinMerge to show the differences.`,
	Args:  cobra.RangeArgs(2, 4),
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error

		if orcaVars.pbVersion != 22 {
			return fmt.Errorf("currently, only PowerBuilder 22 is supported")
		}
		mergeTool, err = filepath.Abs(mergeTool)
		if err != nil {
			return err
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

		pblFilePathBase, err := getCleanPblPbtFilePath(basePath, args[0])
		if err != nil {
			return err
		}
		pblFilePathMine, err := getCleanPblPbtFilePath(basePath, args[1])
		if err != nil {
			return err
		}
		if len(args) == 2 {
			err = diff(Orca, pblFilePathBase, pblFilePathMine)
			if err != nil {
				fmt.Println(err)
			}
			return nil
		}

		if len(args) > 2 {
			pblFilePathTheirs, err := getCleanPblPbtFilePath(basePath, args[2])
			if err != nil {
				return err
			}

			var pblFilePathMerged string
			if len(args) >= 4 {
				pblFilePathMerged, err = getCleanPblPbtFilePath(basePath, args[3])
				if err != nil {
					return err
				}
			}
			if true == false {
				merge(Orca, pblFilePathBase, pblFilePathMine, pblFilePathTheirs, pblFilePathMerged)
			}

			return err
		}
		return nil
	},
}

func diff(Orca *pborca.Orca, objFilePathBase, objFilePathMine string) error {
	tempDir := filepath.Join(os.TempDir(), "pbdiff", time.Now().Format("20060102_150405"))
	os.MkdirAll(tempDir, 0664)
	defer os.RemoveAll(tempDir)
	objSrcPathBase := filepath.Join(tempDir, fmt.Sprintf("%s (%s)", filepath.Base(objFilePathBase), getPblFileDescr(objFilePathBase)))
	os.MkdirAll(objSrcPathBase, 0664)
	objSrcPathMine := filepath.Join(tempDir, fmt.Sprintf("%s (%s)", filepath.Base(objFilePathMine), getPblFileDescr(objFilePathMine)))
	os.MkdirAll(objSrcPathMine, 0664)

	c := make(chan exportJob)

	if filepath.Ext(objFilePathBase) == ".pbt" {
		//producer for pbt
		go func() {
			pbt, _ := orca.NewPbtFromFile(objFilePathBase)
			for _, lib := range pbt.LibList {
				var job exportJob
				job.libraryPath = lib
				job.destinationPath = objSrcPathBase
				c <- job
			}
			pbt, _ = orca.NewPbtFromFile(objFilePathMine)
			for _, lib := range pbt.LibList {
				var job exportJob
				job.libraryPath = lib
				job.destinationPath = objSrcPathMine
				c <- job
			}
			close(c)
		}()
	} else if filepath.Ext(objFilePathBase) == ".pbl" {
		//producer for pbt
		go func() {

			var job exportJob
			job.libraryPath = objFilePathBase
			job.destinationPath = objSrcPathBase
			c <- job

			job.libraryPath = objFilePathMine
			job.destinationPath = objSrcPathMine
			c <- job

			close(c)
		}()
	}

	//consumer
	numOfConsumers := 4
	var wg1 sync.WaitGroup
	for i := 1; i <= numOfConsumers; i++ {
		wg1.Add(1)
		go func() {
			defer wg1.Done()
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
				panic(err)
			}

			for job := range c {
				fmt.Println("Exporting ", job.libraryPath, " to ", job.destinationPath)
				err := exportPbl(Orca, job.libraryPath, regexp.MustCompile("^.*$"), job.destinationPath)
				if err != nil {
					fmt.Println(err)
				}
			}
			Orca.Close()
		}()
	}

	wg1.Wait()
	command := exec.Command(mergeTool, "/r", "/x", "/u", "/ignoreblanklines", objSrcPathMine, objSrcPathBase, "/dl", nameMine, "/dr", nameBase)
	out, err := command.CombinedOutput()
	if err != nil {
		return err
	}
	if len(out) > 0 {
		fmt.Println(out)
	}
	return nil
}

func merge(Orca *pborca.Orca, pblFilePathBase, pblFilePathMine, pblFilePathTheirs, pblFilePathMerged string) error {
	tempDir := filepath.Join(os.TempDir(), "pbdiff", time.Now().Format("20170907_170606"))
	os.MkdirAll(tempDir, 0664)
	defer os.RemoveAll(tempDir)

	pblSrcPathBase := filepath.Join(tempDir, fmt.Sprintf("%s (%s)", filepath.Base(pblFilePathBase), getPblFileDescr(pblFilePathBase)))
	os.MkdirAll(pblSrcPathBase, 0664)
	err := exportPbl(Orca, pblFilePathBase, regexp.MustCompile("^.*$"), pblSrcPathBase)
	if err != nil {
		return err
	}
	pblSrcPathMine := filepath.Join(tempDir, fmt.Sprintf("%s (%s)", filepath.Base(pblFilePathMine), getPblFileDescr(pblFilePathMine)))
	os.MkdirAll(pblSrcPathMine, 0664)
	err = exportPbl(Orca, pblFilePathMine, regexp.MustCompile("^.*$"), pblSrcPathMine)
	if err != nil {
		return err
	}
	pblSrcPathTheirs := filepath.Join(tempDir, fmt.Sprintf("%s (%s)", filepath.Base(pblFilePathTheirs), getPblFileDescr(pblFilePathTheirs)))
	os.MkdirAll(pblSrcPathTheirs, 0664)
	err = exportPbl(Orca, pblFilePathTheirs, regexp.MustCompile("^.*$"), pblSrcPathTheirs)
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
	fmt.Println("Import finished")
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
