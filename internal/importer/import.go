// Package importer imports several source files into several PBLs
package importer

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/informaticon/dev.win.base.pbmanager/utils"
	pborca "github.com/informaticon/lib.go.base.pborca"

	_ "embed"
)

type MultiImport struct {
	backportDir           string
	pbtFilePath           string
	pblFilePaths          []string
	pblSrcFilePaths       []string
	numberWorkers         int
	numberRunningWorkers  *int32
	numberOfMinIterations int
	opts                  []func(*pborca.Orca)
	wg                    *sync.WaitGroup
	errMutex              *sync.Mutex
	errs                  map[string]error
	iteration             int
}

func NewMultiImport(pbtFilePath string, pblFilePaths, pblSrcFilePaths []string,
	options ...func(*MultiImport),
) *MultiImport {
	var runningWorkers int32 = 0
	m := &MultiImport{
		backportDir:           "workspace",
		pbtFilePath:           pbtFilePath,
		pblFilePaths:          pblFilePaths,
		pblSrcFilePaths:       pblSrcFilePaths,
		iteration:             0,
		numberWorkers:         1,
		numberRunningWorkers:  &runningWorkers,
		numberOfMinIterations: 3,
		wg:                    &sync.WaitGroup{},
		errMutex:              &sync.Mutex{},
		errs:                  make(map[string]error),
	}
	for _, opt := range options {
		opt(m)
	}
	return m
}

// WithBackportDir changes the default "workspace" created beside .pbsln or .pbproj.
func WithBackportDir(dir string) func(*MultiImport) {
	return func(multiImport *MultiImport) {
		multiImport.backportDir = dir
	}
}

// WithMinIterations changes the default 3; can be increased for larger projects, each iteration the number of
// compilation errors is typically reduced. E.g. ~10 runs are needed at least for erp.win.base.main
func WithMinIterations(numberIterations int) func(*MultiImport) {
	return func(multiImport *MultiImport) {
		multiImport.numberOfMinIterations = numberIterations
	}
}

// WithNumberWorkers overwrites default 1, meaning no parallelization.
func WithNumberWorkers(numberWorkers int) func(*MultiImport) {
	return func(multiImport *MultiImport) {
		multiImport.numberWorkers = numberWorkers
	}
}

// WithOrcaOpts provides orca server options.
func WithOrcaOpts(orcaOpts []func(*pborca.Orca)) func(*MultiImport) {
	return func(multiImport *MultiImport) {
		multiImport.opts = orcaOpts
	}
}

// Import tries to import into multiple pbls.
// It tries to do it multiple time, so it also works from circular dependencies.
func (m *MultiImport) Import() error {
	minRun := m.numberOfMinIterations
	maxRun := len(m.pblFilePaths) * 3
	lastErrCount := 5000

	for {
		t1 := time.Now()
		m.iteration++
		// collect only errors of one iteration, number should decrease with each iteration
		m.errs = make(map[string]error)
		itemChan := make(chan pblInstance, m.numberWorkers)
		// spawn routines
		for w := 1; w <= m.numberWorkers; w++ {
			m.wg.Add(1)
			atomic.AddInt32(m.numberRunningWorkers, 1)
			go m.worker(w, itemChan)
		}

		for i, pblFilePath := range m.pblFilePaths {
			itemChan <- pblInstance{
				pblFilePath:    pblFilePath,
				pblSrcFilePath: m.pblSrcFilePaths[i],
			}
		}
		close(itemChan)
		m.wg.Wait()
		print("all workers returned\n")
		fmt.Printf("Run %d took %s\n", m.iteration, time.Since(t1).Truncate(time.Second).String())
		m.errMutex.Lock()
		numberErrors := len(m.errs)
		m.errMutex.Unlock()
		if numberErrors == 0 {
			return nil
		}
		if numberErrors >= lastErrCount && m.iteration > minRun {
			return fmt.Errorf("compilation errors occured (multiple tries did not help): %v", m.errs)
		}
		if numberErrors > 0 && m.iteration >= maxRun {
			return fmt.Errorf("compilation errors occured: %v", m.errs)
		}

		lastErrCount = numberErrors
		m.pblSrcFilePaths = append(m.pblSrcFilePaths[1:], m.pblSrcFilePaths[0])
		m.pblFilePaths = append(m.pblFilePaths[1:], m.pblFilePaths[0]) // why is first be placed at last one?
		fmt.Printf("Got %d errors in run %d. Retry...\n", numberErrors, m.iteration)
	}
}

// pblInstance is a pair of initially empty pbl file and related source directory
type pblInstance struct {
	pblFilePath    string
	pblSrcFilePath string
}

func (m *MultiImport) worker(id int, pblInstanceChan chan pblInstance) {
	defer m.wg.Done()
	defer atomic.AddInt32(m.numberRunningWorkers, -1)

	orcaServer, err := pborca.NewOrca(22, pborca.WithOrcaTimeout(time.Minute))
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		slog.Info("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
		orcaServer.Close()
		time.Sleep(1 * time.Second)
	}()

	for item := range pblInstanceChan {
		if m.processPbl(item, orcaServer) {
			return
		}
	}
	fmt.Println("worker", id, "finished")
}

// processPbl exitWorker is used on server crash. All following imports would just empty the pbls, avoid error counter
// increase by this measure.
func (m *MultiImport) processPbl(item pblInstance, orcaServer *pborca.Orca) (exitWorker bool) {
	if filepath.Base(item.pblFilePath) == "pbdom.pbl" {
		m.handlePbDom(item.pblFilePath)
		return false
	}
	fmt.Println("backport", filepath.Base(item.pblFilePath))

	var files []string
	err := filepath.WalkDir(item.pblSrcFilePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			// first collect all files, the order of importing matters, e.g. some.bin after some.srw
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	// get all .bin files, might be several; key filename, val index
	binFiles := make(map[string]int)
	for i, file := range files {
		if filepath.Ext(file) == ".bin" {
			binFiles[strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))] = i
		}
	}
	for _, file := range files {
		if filepath.Ext(file) == ".bin" {
			continue
		}
		_, hasBin := binFiles[strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))]
		err = m.processSrcFile(item.pblFilePath, file, hasBin, orcaServer)
		if m.handleProcessingError(err, file) {
			return true
		}
	}
	return false
}

//go:embed pbdom.pbl
var pbdomPbl []byte // can't be imported by source; TODO: another pbdom.pbl for each project?

// handlePbDom writes an embedded pbdom.pbl instead of importing since it comes from pbdk pbni extension.
func (m *MultiImport) handlePbDom(pblFilePath string) {
	fmt.Println("use embedded pbdom.pbl, skip source import")
	err := os.WriteFile(pblFilePath, pbdomPbl, 0o644)
	if err != nil {
		log.Fatal(err) // it is expected to work
	}
}

// processSrcFile reads the source and if withBin is true it also reads the equally named .bin file and adds its content
// as binary section to the read bytes.
func (m *MultiImport) processSrcFile(pblFilepath, sourcePath string, hasBin bool, orcaServer *pborca.Orca) error {
	objName := strings.TrimSuffix(filepath.Base(sourcePath), filepath.Ext(filepath.Base(sourcePath)))
	srcData, err := utils.ReadPbSource(sourcePath)
	if err != nil {
		return err
	}
	// If .bin counterpart is existent, first import only the source part up to
	// "Start of PowerBuilder Binary Data Section..." as actual object type (pbe_datawindow, pbe_window, ...)
	// In a second step call the same function immediately after containing the binary data part as PBORCA_BINARY.
	// Since bin data part is not real part of source file. ONe can simply use srcData for the first step.
	errSrc := orcaServer.SetObjSource(m.pbtFilePath, pblFilepath, filepath.Base(objName), srcData)
	if hasBin {
		binFile := strings.TrimSuffix(sourcePath, filepath.Ext(sourcePath)) + ".bin"
		binSection, err := GetBinarySectionFromBin(binFile)
		if err != nil {
			return fmt.Errorf("failed to set OLE binary section to matching bin file %s: %v", binFile,
				errors.Join(err, errSrc))
		}
		errBin := orcaServer.SetObjBinary(m.pbtFilePath, pblFilepath, filepath.Base(objName), binSection)
		if errBin != nil {
			return fmt.Errorf("failed to import binary data section in a second step %s: %v", binFile,
				errors.Join(errBin, errSrc))
		}
	}
	return errSrc
}

// handleProcessingError should only exit the worker if there are enough left to process the remaining PBLs else
// a deadlock will occur. If the error indicates an orca server crash and the number of currently active workers is >1
// then the affected worker must be shut down so that the remaining source can be imported without trivial processing,
// i.e. clearing the pbls and increasing the error counter drastically.
// If only one worker is operational the program can exit.
// The reason for the server crash
func (m *MultiImport) handleProcessingError(err error, file string) (shutdownWorker bool) {
	if err == nil {
		return false
	}
	m.errMutex.Lock()
	m.errs[filepath.Base(file)] = err
	m.errMutex.Unlock()
	// those compilation errors are expected and reduce with each iteration
	if !strings.Contains(err.Error(), "Compilation failed") {
		log.Print(filepath.Base(file), ": ", err.Error())
	}
	// there are also errors when the server crashes: "wsarecv: ..." but the reason is not yet known. In this case
	// all following objects of this pbl are dummy processed increasing the number of errors.
	// Stop whole process with error (what object caused server crash) if there is only one worker left. Else
	// shut down this worker, and continue with next source file
	if strings.Contains(err.Error(), "wsarecv:") {
		if atomic.LoadInt32(m.numberRunningWorkers) <= 1 {
			log.Fatalf("processing import of %s, only one worker is running, "+
				"connection to orca server is gone: %v; %s might have caused server crash", file, err, file)
		} else {
			log.Printf("exit worker: %s caused server crash: %v", file, err)
			return true
		}
	}
	return false
}

// writeStageResultTemp writes for debug purpose for a certain run number and file types (those with errors)
// write the source before processing and after so one can compare bytes (e.g. temp)
func writeStageResultTemp(pblFilepath string, srcData []byte) {
	tempDir, errTemp := os.MkdirTemp("", fmt.Sprintf("backport_%s", filepath.Base(pblFilepath)))
	if errTemp != nil {
		log.Println(errTemp)
	}
	fmt.Printf("write %s and source after import operation into separate copy %s\n",
		filepath.Base(pblFilepath), tempDir)
	// SOURCE file
	errTemp = os.WriteFile(filepath.Join(tempDir, filepath.Base(pblFilepath)+"_source"), srcData, 0o644)
	if errTemp != nil {
		log.Println(errTemp)
	}
	// PBL
	pblAfterImportStep, errTemp := os.ReadFile(pblFilepath)
	errTemp = os.WriteFile(filepath.Join(tempDir, filepath.Base(pblFilepath)+"_imported"),
		pblAfterImportStep, 0o644)
	if errTemp != nil {
		log.Println(errTemp)
	}
}
