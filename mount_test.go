package fsmount

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"
)

func TestMountFS(t *testing.T) {
	targetDir := "."
	mountPoint, err := os.MkdirTemp("", "testmount")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(mountPoint)
	t.Log("Mount point:", mountPoint)

	mount, err := MountFS(mountPoint, os.DirFS(targetDir), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer mount.Close()
	time.Sleep(10 * time.Millisecond)

	fname := filepath.Join(mountPoint, "LICENSE")

	stat, err := os.Stat(fname)
	if err != nil {
		t.Fatal("Stat() error", err)
	}
	t.Log("Name: ", stat.Name())
	t.Log("Size: ", stat.Size())
	t.Log("ModTime: ", stat.ModTime())
	t.Log("IsDir: ", stat.IsDir())
	t.Log("Mode: ", stat.Mode())

	r, err := os.Open(fname)
	if err != nil {
		t.Fatal("Open() error", err)
	}

	_, err = r.Write([]byte("Test"))
	if err == nil {
		t.Error("Write() should be failed")
	}

	buf := make([]byte, 32)
	_, err = r.Read(buf)
	if err != nil {
		t.Error("Read() error", err)
	}

	_, err = r.ReadAt(buf, 8)
	if err != nil {
		t.Error("Read() error", err)
	}

	err = r.Close()
	if err != nil {
		t.Error("Close() error", err)
	}

	r, err = os.Open(mountPoint + "/notfound")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Error("Open() for not exitst file should be failed.", err)
	}

	r, err = os.OpenFile(mountPoint+"/notfound", os.O_CREATE|os.O_WRONLY, 0)
	if !errors.Is(err, fs.ErrPermission) {
		t.Error("OpenFile() wtih O_CREATE should be failed.", err)
	}

	b, err := os.ReadFile(fname)
	if err != nil {
		t.Error("ReadFile() error", err)
	}
	t.Log("Content: ", string(b))

}

type testWritableFs struct {
	fs.FS
	path string
}

func (fsys *testWritableFs) OpenWriter(name string, flag int) (io.WriteCloser, error) {
	return os.OpenFile(path.Join(fsys.path, name), flag, fs.ModePerm)
}

func (fsys *testWritableFs) Remove(name string) error {
	return os.Remove(path.Join(fsys.path, name))
}

func (fsys *testWritableFs) Mkdir(name string, mode fs.FileMode) error {
	return os.Mkdir(path.Join(fsys.path, name), mode)
}

func (fsys *testWritableFs) Rename(name, newName string) error {
	return os.Rename(path.Join(fsys.path, name), path.Join(fsys.path, newName))
}

func TestWritableFS(t *testing.T) {

	targetDir := "testdata"
	mountPoint, err := os.MkdirTemp("", "testmount")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(mountPoint)
	t.Log("Mount point:", mountPoint)

	mount, err := MountFS(mountPoint, &testWritableFs{FS: os.DirFS(targetDir), path: targetDir}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer mount.Close()
	time.Sleep(10 * time.Millisecond)

	fname := filepath.Join(mountPoint, "output.txt")

	_ = os.Remove(fname)
	_ = os.Remove(fname + ".renamed")

	files, err := os.ReadDir(mountPoint)
	if err != nil {
		t.Fatal("ReadDir() error", err)
	}
	t.Log("ReadDir() files: ", len(files))

	f, err := os.Create(fname)
	if err != nil {
		t.Fatal("Create() error", err)
	}

	_, err = f.Write([]byte("hello, FUSE!\n"))
	if err != nil {
		t.Fatal("Write() error", err)
	}
	_, err = f.Write([]byte("1234567890"))
	if err != nil {
		t.Fatal("Write() error", err)
	}

	_, err = f.WriteAt([]byte("Hello"), 0)
	if err != nil {
		t.Fatal("WriteAt() error", err)
	}

	err = f.Close()
	if err != nil {
		t.Fatal("Close() error", err)
	}

	err = os.Truncate(fname, 1)
	if err != nil {
		t.Error("Truncate() error", err)
	}

	err = os.Rename(fname, fname+".renamed")
	if err != nil {
		t.Fatal("Rename() error", err)
	}

	err = os.Remove(fname + ".renamed")
	if err != nil {
		t.Fatal("Remove() error", err)
	}

	dname := filepath.Join(mountPoint, "dir")

	err = os.Mkdir(dname, fs.ModePerm)
	if err != nil {
		t.Fatal("Mkdir() error", err)
	}

	err = os.Remove(dname)
	if err != nil {
		t.Fatal("Remove() dir error", err)
	}

	f, err = os.OpenFile(fname, os.O_WRONLY, os.ModePerm)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatal("OpenFile() should fail with ErrNotExist", err)
	}

	// Create empty file
	f, err = os.Create(fname)
	if err != nil {
		t.Fatal("Create() error", err)
	}
	err = f.Close()
	if err != nil {
		t.Fatal("Close() error", err)
	}

	// Append1
	f, err = os.OpenFile(fname, os.O_WRONLY|os.O_APPEND, os.ModePerm)
	if err != nil {
		t.Fatal("OpenFile() error", err)
	}
	_, err = f.Write([]byte("01234"))
	if err != nil {
		t.Error("Write() error", err)
	}
	err = f.Close()
	if err != nil {
		t.Fatal("Close() error", err)
	}

	// Append2
	f, err = os.OpenFile(fname, os.O_WRONLY|os.O_APPEND, os.ModePerm)
	if err != nil {
		t.Fatal("OpenFile() error", err)
	}
	_, err = f.Write([]byte("56789"))
	if err != nil {
		t.Error("Write() error", err)
	}

	err = f.Close()
	if err != nil {
		t.Fatal("Close() error", err)
	}

	stat, err := os.Stat(fname)
	if stat.Size() != 10 {
		t.Error("Size() should returns 10", stat.Size())
	}

	// Err if exists
	f, err = os.OpenFile(fname, os.O_WRONLY|os.O_CREATE|os.O_EXCL, os.ModePerm)
	if !errors.Is(err, fs.ErrExist) {
		t.Error("OpenFile() should fail with ErrExist", err)
	}

	err = os.Remove(fname)
	if err != nil {
		t.Error("Remove() error", err)
	}
}
