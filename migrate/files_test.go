package migrate

import (
	"fmt"
	"slices"
	"testing"
)

func TestCheckForUncommonFiles(t *testing.T) {
	ret, err := CheckForUncommonFiles("testdata")
	if err != nil {
		t.Fatal(err)
	}
	if len(ret) != 2 || !slices.Contains(ret, "testdata\\pbdk\\dispo.png") || !slices.Contains(ret, "testdata\\pbdk\\dispo.jpeg") {
		t.Fatal(fmt.Sprintf("Uncommon files were not detected correctls: %s", ret))
	}
}
