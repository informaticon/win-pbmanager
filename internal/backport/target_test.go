package backport

import (
	"reflect"
	"testing"
)

// TestParseName checks the name and suffix parsing logic.
func TestParseName(t *testing.T) {
	testCases := []struct {
		input     string
		expBase   string
		expSuffix int
	}{
		{"xyz3", "xyz", 3},
		{"abc2", "abc", 2},
		{"ghf1", "ghf", 1},
		{"nosuffix", "nosuffix", 0},
		{"itemwith1", "itemwith", 1}, // Make sure it parses from the end
		{"item4", "item4", 0},        // '4' is not a recognized suffix
		{"", "", 0},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			base, suffix := parseName(tc.input)
			if base != tc.expBase || suffix != tc.expSuffix {
				t.Errorf("parseName(%q) = (%q, %d), want (%q, %d)",
					tc.input, base, suffix, tc.expBase, tc.expSuffix)
			}
		})
	}
}

// TestGetPriority checks the list-based priority assignment.
func TestGetPriority(t *testing.T) {
	listMap := buildListMap(
		[]string{"apple", "zebra"}, // list 1 -> prio 1
		[]string{"dog"},            // list 2 -> prio 2
		[]string{"star", "moon"},   // list 3 -> prio 3
	)

	testCases := []struct {
		input   string
		expPrio int
	}{
		{"apple", 1},     // List 1
		{"zebra", 1},     // List 1
		{"dog", 2},       // List 2
		{"star", 3},      // List 3
		{"unlisted", 11}, // Not in any list
		{"other", 11},    // Not in any list
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			prio := getPriority(tc.input, listMap)
			if prio != tc.expPrio {
				t.Errorf("getPriority(%q) = %d, want %d", tc.input, prio, tc.expPrio)
			}
		})
	}
}

// TestFullSort runs comprehensive tests on the sorting logic.
func TestFullSort(t *testing.T) {
	// Define the lists once for all test cases
	list1 := []string{"zebra", "apple"}
	list2 := []string{"dog"}
	list3 := []string{}
	list4 := []string{"house"}
	list5 := []string{"gamma", "beta"}
	list6 := []string{"jupiter"}
	list7 := []string{"phone"}
	list8 := []string{"water"}
	list9 := []string{"carrot", "banana"}
	list10 := []string{"star", "moon"}

	listMap := buildListMap(
		list1, list2, list3, list4, list5,
		list6, list7, list8, list9, list10,
	)

	target := &Target{ListMap: listMap}

	// Define all test cases
	testCases := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "Main Example from Prompt",
			input:    []string{"apple1", "star3", "unlisted2", "banana", "zebra3", "moon2", "unlisted1", "gamma3", "star1", "beta1", "zebra1", "other_item", "moon1", "carrot2", "unlisted3", "beta2"},
			expected: []string{"other_item", "unlisted3", "unlisted2", "unlisted1", "moon2", "moon1", "star3", "star1", "banana", "carrot2", "beta2", "beta1", "gamma3", "apple1", "zebra3", "zebra1"},
		},
		{
			name:     "Rule 1: Unlisted Items First",
			input:    []string{"apple1", "unlisted2", "star3", "unlisted1", "zebra1", "unlisted3"},
			expected: []string{"unlisted3", "unlisted2", "unlisted1", "star3", "apple1", "zebra1"},
		},
		{
			name:     "Rule 2: Alphabetical (Same List)",
			input:    []string{"zebra1", "apple1"}, // Both list 1
			expected: []string{"apple1", "zebra1"},
		},
		{
			name:     "Rule 3: Suffix Priority (Same Base)",
			input:    []string{"zebra1", "zebra3", "zebra2"}, // All list 1
			expected: []string{"zebra3", "zebra2", "zebra1"},
		},
		{
			name:     "List Priority (List 10 vs List 1)",
			input:    []string{"apple1", "star1"}, // list 1 vs list 10
			expected: []string{"star1", "apple1"},
		},
		{
			name:     "Empty Input",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "Single Item",
			input:    []string{"one_item"},
			expected: []string{"one_item"},
		},
		{
			name:     "No Suffixes",
			input:    []string{"apple", "star", "unlisted", "zebra", "moon"},
			expected: []string{"unlisted", "moon", "star", "apple", "zebra"},
		},
		{
			name:     "Complex Mix",
			input:    []string{"beta1", "beta3", "alpha", "star1", "star2", "apple3"},
			expected: []string{"alpha", "star2", "star1", "beta3", "beta1", "apple3"},
			// "alpha" (unlisted, prio 11)
			// "star2", "star1" (list 10, prio 10)
			// "beta3", "beta1" (list 5, prio 5)
			// "apple3" (list 1, prio 1)
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			target.LibList = tc.input
			target.SortLibList()
			actual := target.LibList
			if !reflect.DeepEqual(actual, tc.expected) {
				t.Errorf("Test %q failed:\nInput:    %v\nExpected: %v\nActual:   %v",
					tc.name, tc.input, tc.expected, actual)
			}
		})
	}
}

func TestTarget_SortLibList(t *testing.T) {
	realLibList := []string{
		"arf1.pbl", "bai1.pbl", "cfg1.pbl", "inf1.pbl", "kal1.pbl", "nfy1.pbl", "pbdom.pbl",
		"sti1.pbl", "stm1.pbl", "tse1.pbl", "adr1.pbl", "arc1.pbl", "art1.pbl", "bde1.pbl", "biz1.pbl", "dgm1.pbl",
		"dis1.pbl", "drucken1.pbl", "dzb1.pbl", "ecp1.pbl", "ein1.pbl", "ger1.pbl", "kon1.pbl", "lag1.pbl", "lda1.pbl",
		"map1.pbl", "mit1.pbl", "obj1.pbl", "ord1.pbl", "pos1.pbl", "prj1.pbl", "rap1.pbl", "res1.pbl", "sal1.pbl",
		"anl1.pbl", "deb1.pbl", "fib1.pbl", "fin1.pbl", "fre1.pbl", "kor1.pbl", "kre1.pbl", "mai1.pbl", "mve1.pbl",
		"tbs1.pbl", "zea1.pbl", "zei1.pbl", "zek1.pbl", "zes1.pbl", "kim1.pbl", "eft1.pbl", "exf1.pbl", "fsu1.pbl",
		"grp1.pbl", "liq1.pbl", "net1.pbl", "osu1.pbl", "str1.pbl", "szn1.pbl", "jif1.pbl", "lif1.pbl",
	}
	expectedLibList := []string{
		"adr1.pbl", "arc1.pbl", "art1.pbl", "bde1.pbl", "biz1.pbl", "dgm1.pbl", "dis1.pbl",
		"drucken1.pbl", "dzb1.pbl", "ecp1.pbl", "ein1.pbl", "fre1.pbl", "ger1.pbl", "kon1.pbl", "lag1.pbl", "lda1.pbl",
		"map1.pbl", "mit1.pbl", "obj1.pbl", "ord1.pbl", "pos1.pbl", "prj1.pbl", "rap1.pbl", "res1.pbl", "sal1.pbl",
		"anl1.pbl", "deb1.pbl", "fib1.pbl", "fin1.pbl", "kor1.pbl", "kre1.pbl", "mai1.pbl", "mve1.pbl", "tbs1.pbl",
		"zea1.pbl", "zei1.pbl", "zek1.pbl", "zes1.pbl", "bai1.pbl", "kim1.pbl", "eft1.pbl", "exf1.pbl", "fsu1.pbl",
		"grp1.pbl", "liq1.pbl", "net1.pbl", "osu1.pbl", "str1.pbl", "szn1.pbl", "arf1.pbl", "cfg1.pbl", "inf1.pbl",
		"kal1.pbl", "nfy1.pbl", "pbdom.pbl", "sti1.pbl", "stm1.pbl", "tse1.pbl", "jif1.pbl", "lif1.pbl",
	}
	target := NewTarget("a3", "inf1.pbl", realLibList)
	target.SortLibList()
	actual := target.LibList
	if !reflect.DeepEqual(actual, expectedLibList) {
		t.Errorf("Test %q failed:\nInput:    %v\nExpected: %v\nActual:   %v",
			"real world example", realLibList, expectedLibList, actual)
	}
}
