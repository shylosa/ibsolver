package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteExchangeReplacesOnlyCompleteFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "export.json")
	for _, data := range [][]byte{[]byte("first"), []byte("second complete export")} {
		if err := writeExchange(path, data); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(got, data) {
			t.Fatal("export differs", err)
		}
	}
	parent := filepath.Dir(path)
	if err := writeExchange(parent, []byte("must fail")); err == nil {
		t.Fatal("replaced directory")
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "second complete export" {
		t.Fatal("failed export changed previous file", err)
	}
	files, err := os.ReadDir(parent)
	if err != nil || len(files) != 1 {
		t.Fatal("temporary file leaked", err)
	}
}
