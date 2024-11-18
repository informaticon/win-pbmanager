package utils

import (
	"fmt"
	"testing"
)

func TestGetCommonBaseDir(t *testing.T) {
	fmt.Println(GetCommonBaseDir("C:\\a3\\test\\123", "C:/a3/test/fest/aaa"))
}

func TestReadPbSrc(t *testing.T) {
	cases := []struct {
		file     string
		expected string
	}{
		{file: "testdata/utf8.txt", expected: "$PBExportHeader$utf8.txt\r\nHello World"},
		{file: "testdata/utf8bom.txt", expected: "$PBExportHeader$utf8bom.txt\r\nHello World"},
		{file: "testdata/utf16le.txt", expected: "$PBExportHeader$utf16le.txt\r\nHello World"},
		{file: "testdata/utf16be.txt", expected: "$PBExportHeader$utf16be.txt\r\nHello World"},
	}
	for _, cas := range cases {
		got, err := ReadPbSource(cas.file)
		if err != nil {
			t.Error(err)
			continue
		}
		if got != cas.expected {
			t.Errorf("%s content is wrong. Expected: %s, Got: %s", cas.file, cas.expected, got)
		}
	}
}
