package main

import (
	"hash/crc32"
	"os"
	"testing"
)

func TestJTF(t *testing.T) {
	b, err := os.ReadFile("comment.jtf")
	if err != nil {
		t.Fatal(err)
	}
	want := crc32.ChecksumIEEE(b)

	b, err = os.ReadFile("comment.md")
	if err != nil {
		t.Fatal(err)
	}
	rendered := toJTF(string(b))
	got := crc32.ChecksumIEEE([]byte(rendered))
	if want != got {
		t.Errorf("unexepected content in rendered comment")
		t.Logf("%s\n", string(rendered))
	}
}
