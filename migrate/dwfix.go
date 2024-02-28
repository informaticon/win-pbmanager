package migrate

import (
	"fmt"
	"regexp"

	pborca "github.com/informaticon/lib.go.base.pborca"
	"github.com/informaticon/lib.go.base.pborca/orca"
)

// FixDatawindows moves datawindow checkboxes from centered to left aligned
func FixDatawindows(pbtData *orca.Pbt, o *pborca.Orca, warnFunc func(string)) error {
	var errs []string
	for _, pbl := range pbtData.LibList {
		objs, err := o.GetObjList(pbl)
		if err != nil {
			errs = append(errs, fmt.Sprintf("could not export %s: %v", pbl, err))
			continue
		}
		for _, objArr := range objs {
			for _, obj := range objArr.GetObjArr() {
				if obj.ObjType != orca.ObjType_DATAWINDOW {
					continue
				}
				src, err := o.GetObjSource(pbl, obj.Name+".srd")
				if err != nil {
					errs = append(errs, fmt.Sprintf("could not get source of %s in %s: %v", obj.Name, pbl, err))
					continue
				}
				changed1, src := fixCheckboxAlignment(src)
				changed2, src := fixHorizontalScrollbar(src)
				if changed1 || changed2 {
					fmt.Printf("Fix Dw %s because of %s\n", obj.Name, func() string {
						if changed1 && changed2 {
							return "CheckboxAlignment and HorizontalScrollbar"
						} else if changed1 {
							return "CheckboxAlignment"
						} else {
							return "HorizontalScrollbar"
						}
					}())
					err = o.SetObjSource(pbtData.GetPath(), pbl, obj.Name, src)
					if err != nil {
						errs = append(errs, fmt.Sprintf("could not write source of %s in %s: %v", obj.Name, pbl, err))
					}
				}
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("one or more error occured in FixDatawindows: %s", errs)
	}
	return nil
}

var regexDwCheckBoxAlign = regexp.MustCompile(`(?im)^(column.*alignment=")2(".*checkbox\.text="[^"])`)

func fixCheckboxAlignment(src string) (bool, string) {
	if !regexDwCheckBoxAlign.MatchString(src) {
		return false, src
	}
	return true, regexDwCheckBoxAlign.ReplaceAllString(src, `${1}0${2}`)
}

var regexDwHorizontalScollbar = regexp.MustCompile(`(?im)^(column.* height="(?:[0-7][0-2]|[0-6][0-9])".* )edit.hscrollbar=yes `)

func fixHorizontalScrollbar(src string) (bool, string) {
	if !regexDwHorizontalScollbar.MatchString(src) {
		return false, src
	}
	return true, regexDwHorizontalScollbar.ReplaceAllString(src, `${1}`)
}
