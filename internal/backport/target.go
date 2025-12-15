package backport

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"
)

// Target contains basic components to create a .pbt file. There might be others that are not necessary for compilation.
type Target struct {
	AppName string
	AppLib  string
	LibList []string
	ListMap map[string]int // fast lookup map for sorting
}

// NewTarget returns the minimal structure of needed for a target file. Expects a list of pbl names, e.g.
// []string{"some_app.pbl", "some_lib.pbl", ...}.
func NewTarget(appName, appEntryPbl string, libList []string) *Target {
	// Defines the 10 hardcoded lists (with base names)
	list1 := []string{"lif"}
	list2 := []string{"jif"}
	list3 := []string{"arf", "cfg", "inf", "kal", "nfy", "pbdom", "sti", "stm", "tse"}
	list4 := []string{"dss", "eft", "exf", "fsu", "grp", "liq", "net", "osu", "str", "szn"}
	list5 := []string{"sfi"}
	list6 := []string{"bai", "kim"}
	list7 := []string{"cfg_lohn", "elm", "elmg", "elmp", "elx", "loh", "lor", "spe", "stm_lohn"}
	list8 := []string{"anl", "deb", "fib", "fin", "fre", "kor", "kre", "mai", "mve", "tbs", "zea", "zei", "zek", "zes"}
	list9 := []string{
		"adr", "arc", "art", "bde", "biz", "dgm", "dis", "drucken", "dzb", "ecp", "ein", "ger",
		"kon", "lag", "lda", "map", "mit", "obj", "ord", "pos", "prj", "rap", "res", "sal",
	}
	list10 := []string{
		"avd", "bbp", "bst", "con", "das", "dbe", "dka", "dma", "dmi", "dta", "dto", "dwh", "dws",
		"egm", "kas", "kat", "kng", "mie", "pro", "reg", "sdi", "ser", "sfm", "wae", "wgb", "wss",
	}

	return &Target{
		AppName: appName,
		AppLib:  appEntryPbl,
		LibList: libList,
		ListMap: buildListMap(
			list1, list2, list3, list4, list5,
			list6, list7, list8, list9, list10,
		),
	}
}

// ToBytes must cut off the library list paths and can only consider the file name, e.g. not test\\exf1.pbl but only
// exf1.pbl.
// This is needed since else the backporting is not working properly as pbautobuild220.exe expects a flat pbl structure
// within ws_objects and beside the target. Else an error "application not set" occurs.
func (t *Target) ToBytes() []byte {
	for i, lib := range t.LibList {
		t.LibList[i] = filepath.Base(lib)
	}
	t.SortLibList()

	t.AppLib = filepath.Base(t.AppLib)

	targetString := fmt.Sprintf(`Save Format v3.0(19990112)
appname "%s";
applib "%s";
liblist "%s";
type "pb";`,
		t.AppName,
		strings.ReplaceAll(filepath.ToSlash(t.AppLib), "/", "\\\\"),
		strings.ReplaceAll(strings.ReplaceAll(strings.Join(t.LibList, ";"), "\\", "/"), "/", "\\\\"),
	)
	return []byte(targetString)
}

// parseName splits an item like "xyz3" into its base ("xyz") and suffix (3).
// Items with no suffix are given a suffix priority of 0.
func parseName(item string) (baseName string, suffix int) {
	if strings.HasSuffix(item, "3") {
		return item[:len(item)-1], 3
	}
	if strings.HasSuffix(item, "2") {
		return item[:len(item)-1], 2
	}
	if strings.HasSuffix(item, "1") {
		return item[:len(item)-1], 1
	}
	// No recognized suffix
	return item, 0
}

// buildListMap creates a map for fast lookup of an item's list priority.
// map[baseName] -> priority (1 for list_1, 10 for list_10)
func buildListMap(lists ...[]string) map[string]int {
	listMap := make(map[string]int)
	for i, list := range lists {
		// Assign priority based on list index (0-9)
		// list_1 (index 0) -> priority 1
		// list_10 (index 9) -> priority 10
		priority := i + 1
		for _, item := range list {
			listMap[item] = priority
		}
	}
	return listMap
}

// getPriority returns the sorting priority for a given base name.
// Unlisted items get the highest priority (11) to appear first.
func getPriority(baseName string, listMap map[string]int) int {
	if priority, ok := listMap[baseName]; ok {
		return priority // 1-10
	}
	// "Unlisted" items get the highest priority so they sort first.
	return 11
}

// SortLibList sorts according to defined excel "Packages und Verantwortlichkeiten" (10: first packages, ...,
// 1:), their number 3,2,1 and alphabetically within the same list.
func (t *Target) SortLibList() {
	sort.Slice(t.LibList, func(i, j int) bool {
		a := strings.TrimSuffix(t.LibList[i], ".pbl")
		b := strings.TrimSuffix(t.LibList[j], ".pbl")

		// get pbl base names and suffix priorities (3,2,1)
		baseA, suffixA := parseName(a)
		baseB, suffixB := parseName(b)

		// Get list priorities
		prioA := getPriority(baseA, t.ListMap)
		prioB := getPriority(baseB, t.ListMap)

		// Rule 1: List Priority (11 > 10 > ... > 1)
		if prioA != prioB {
			return prioA > prioB
		}

		// Rule 2: Alphabetical by Base Name
		if baseA != baseB {
			return baseA < baseB
		}

		// Rule 3: Suffix Priority (3 > 2 > 1 > 0)
		return suffixA > suffixB
	})

	slog.Debug("--- Sorted List ---")
	slog.Debug(fmt.Sprintf("%s", t.LibList))
}
