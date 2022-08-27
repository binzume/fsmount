package fsmount

import (
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestMount(t *testing.T) {
	fsys := os.DirFS("./testdata")
	close, err := MountFS("X:", fsys, nil)
	if err != nil {
		t.Fatalf("MountFS() error: %v", err)
	}

	time.Sleep(1 * time.Second)

	data, err := ioutil.ReadFile("X:/hello.txt")
	if err != nil {
		t.Errorf("error: %v", err)
	}

	if string(data) != "Hello" {
		t.Errorf("unexpected: %v", string(data))
	}

	time.Sleep(5 * time.Second)

	err = close.Close()
	if err != nil {
		t.Errorf("Close() error: %v", err)
	}
}
