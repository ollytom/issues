package main

import (
	"encoding/json"
	"os"
	"testing"
)

func TestDecode(t *testing.T) {
	dents, err := os.ReadDir("testdata/issue")
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range dents {
		f, err := os.Open("testdata/issue/"+d.Name())
		if err != nil {
			t.Fatal(err)
		}
		var i Issue
		if err := json.NewDecoder(f).Decode(&i); err != nil {
			t.Errorf("decode %s: %v", f.Name(), err)
		}
		f.Close()
	}
}
