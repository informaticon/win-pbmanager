package migrate

import (
	"slices"
	"testing"
)

func TestCheckForUncommonFiles(t *testing.T) {
	ret, err := CheckForUncommonFiles("testdata")
	if err != nil {
		t.Fatal(err)
	}
	if len(ret) != 2 || !slices.Contains(ret, "testdata\\pbdk\\dispo.png") || !slices.Contains(ret, "testdata\\pbdk\\dispo.jpeg") {
		t.Fatalf("Uncommon files were not detected correctls: %s", ret)
	}
}
