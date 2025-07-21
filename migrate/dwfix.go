package migrate

import (
	"fmt"
	"regexp"
	"strings"

	pborca "github.com/informaticon/lib.go.base.pborca"
	"github.com/informaticon/lib.go.base.pborca/orca"
)

var DwfixAll = []func(string) (bool, string, string){
	fixCheckboxAlignment,
	fixHorizontalScrollbar,
	fixHorizontalScrollbarFin,
}
var DwfixFinScrollbar = []func(string) (bool, string, string){fixHorizontalScrollbarFin}

// FixDatawindows moves datawindow checkboxes from centered to left aligned
func FixDatawindows(pbtData *orca.Pbt, o *pborca.Orca, fncs []func(string) (bool, string, string), warnFunc func(string)) error {
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
				msgs := ""
				for _, fnc := range fncs {
					var changed bool
					var msg string
					changed, src, msg = fnc(src)
					if changed {
						if msgs != "" {
							msgs += " and "
						}
						msgs += msg
					}
				}
				if msgs != "" {
					fmt.Printf("Fix Dw %s because of %s\n", obj.Name, msgs)
					err = o.SetObjSource(pbtData.GetPath(), pbl, obj.Name, []byte(src))
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

func fixCheckboxAlignment(src string) (bool, string, string) {
	if !regexDwCheckBoxAlign.MatchString(src) {
		return false, src, "CheckboxAlign"
	}
	return true, regexDwCheckBoxAlign.ReplaceAllString(src, `${1}0${2}`), "CheckboxAlign"
}

var (
	regexDwHorizontalScollbar1 = regexp.MustCompile(`(?im)^(column.* height="(?:[0-7][0-2]|[0-6][0-9])".* edit.autohscroll=yes.* )edit.hscrollbar=yes `)
	regexDwHorizontalScollbar2 = regexp.MustCompile(`(?im)^(column.* height="(?:[0-7][0-2]|[0-6][0-9])".* )edit.hscrollbar=yes `)
)

func fixHorizontalScrollbar(src string) (bool, string, string) {
	ret := false
	if regexDwHorizontalScollbar1.MatchString(src) {
		ret = true
		src = regexDwHorizontalScollbar1.ReplaceAllString(src, `${1}`)
	}
	if regexDwHorizontalScollbar2.MatchString(src) {
		ret = true
		src = regexDwHorizontalScollbar2.ReplaceAllString(src, `${1}edit.autohscroll=yes `)
	}
	return ret, src, "HorizontalScroll"
}

// fixHorizontalScrollbarFin is a subset of FixDatawindows for projects which were already migrated
// with an older pbmanager and therfore can't be fixed with FixDatawindows
func fixHorizontalScrollbarFin(src string) (bool, string, string) {
	dws := map[string]string{
		"d_deb_rechnung":                       "do_stamm_d1_belegnummer",
		"d_deb_mahnung_offene_rechnungen_list": "kontaktprotokoll_insert",
		"d_fin_mwstsatz":                       "mwst_bez",
	}
	for dw, col := range dws {
		if strings.HasPrefix(src, "$PBExportHeader$"+dw) {
			regex := regexp.MustCompile(`(?im)^(column.* name=` + col + ` .* )( font\.face)`)
			if regex.MatchString(src) {
				return true, regex.ReplaceAllString(src, `${1}edit.autohscroll=yes${2}`), "HorizontalScrollFin"
			}
		}
	}
	return false, src, "HorizontalScrollFin"
}
