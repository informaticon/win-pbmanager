package backport

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"

	"github.com/informaticon/dev.win.base.pbmanager/internal/importer"
)

type PbProject struct { // avoid collision with pbsln Project xml entry
	XMLName     xml.Name `xml:"Project"`
	Type        Type
	Application Application // e.g. <Application Name="a3"/>
	Libraries   Libraries
	Opts        []func(*importer.MultiImport)
}

type Type struct {
	Name string `xml:"Name,attr"`
}

type Application struct {
	Name string `xml:"Name,attr"`
}

type Libraries struct {
	XMLName   xml.Name  `xml:"Libraries"`
	AppEntry  string    `xml:"AppEntry,attr"` // e.g. <Libraries AppEntry="inf1.pbl">
	Libraries []Library `xml:"Library"`
}

// GetPblPaths returns a slice of the pbls relative to projects files, e.g. ../lib/grp1.pbl
func (l Libraries) GetPblPaths() []string {
	libs := []string{}
	for _, item := range l.Libraries {
		libs = append(libs, filepath.ToSlash(item.Path))
	}
	return libs
}

type Library struct {
	Path string `xml:"Path,attr"` // e.g. <Library Path="lto2.pbl"/>
}

func NewProject(pbProjFile string, opts []func(*importer.MultiImport)) (*PbProject, error) {
	data, err := os.ReadFile(pbProjFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read project file %s: %v", pbProjFile, err)
	}
	p := &PbProject{}
	err = xml.Unmarshal(data, p)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal project file to structure %s: %v", pbProjFile, err)
	}
	p.Opts = opts
	return p, nil
}

func (p *PbProject) String() string {
	return p.Application.Name
}
