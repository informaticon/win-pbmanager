package utils

import (
	"fmt"
	"testing"
)

func TestGetCommonBaseDir(t *testing.T) {
	fmt.Println(GetCommonBaseDir("C:\\a3\\test\\123", "C:/a3/test/fest/aaa"))
}
