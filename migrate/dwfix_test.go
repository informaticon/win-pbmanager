package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/informaticon/dev.win.base.pbmanager/utils"
	pborca "github.com/informaticon/lib.go.base.pborca"
	"github.com/informaticon/lib.go.base.pborca/orca"
)

func TestFixDatawindows(t *testing.T) {
	pbtData, err := orca.NewPbtFromFile(filepath.Join("testdata/dwfix/dwfix.pbt"))
	if err != nil {
		t.Fatal(err)
	}

	for _, pbl := range pbtData.LibList {
		utils.CopyFile(pbl+".vanilla.pbl", pbl)
		defer os.Remove(pbl)
	}

	o, err := pborca.NewOrca(22)
	if err != nil {
		t.Fatal(err)
	}
	defer o.Close()

	err = FixDatawindows(pbtData, o, printWarn)
	if err != nil {
		t.Fatal(err)
	}

	for _, pbl := range pbtData.LibList {
		objs, err := o.GetObjList(pbl)
		if err != nil {
			t.Fatal(err)
		}
		for _, objArr := range objs {
			for _, obj := range objArr.GetObjArr() {
				if obj.ObjType != orca.ObjType_DATAWINDOW {
					continue
				}
				new, err := o.GetObjSource(pbl, obj.GetName()+".srd")
				if err != nil {
					t.Fatal(err)
				}
				want, err := o.GetObjSource(pbl+".want.pbl", obj.GetName()+".srd")
				if err != nil {
					t.Fatal(err)
				}
				if new != want {
					fmt.Errorf("obj %s wasn't changed as expected", obj.GetName())
				}
			}
		}
	}
}

func printWarn(message string) {
	fmt.Println("WARN: ", message)
}
