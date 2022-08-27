//go:build !windows
// +build !windows

package fsmount

import (
	"errors"
	"io"
	"io/fs"

	"fmt"
	"os"
	"os/signal"

	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/hanwen/go-fuse/v2/fuse/nodefs"
	"github.com/hanwen/go-fuse/v2/fuse/pathfs"
)

func readAt(f io.Closer, b []byte, off int64) (int, error) {
	if f, ok := f.(io.ReaderAt); ok {
		return f.ReadAt(b, off)
	}
	if f, ok := f.(io.ReadSeeker); ok {
		_, err := f.Seek(off, io.SeekStart)
		if err != nil {
			return 0, err
		}
		return io.ReadFull(f, b)
	}
	return 0, fs.ErrInvalid
}

func writeAt(f io.Closer, b []byte, off int64) (int, error) {
	if f, ok := f.(io.WriterAt); ok {
		return f.WriteAt(b, off)
	}
	if f, ok := f.(io.WriteSeeker); ok {
		_, err := f.Seek(off, io.SeekStart)
		if err != nil {
			return 0, err
		}
		return f.Write(b)
	}
	return 0, fs.ErrInvalid
}

func truncate(fsys fs.FS, name string, size int64) error {
	s, ok := fsys.(interface {
		Truncate(string, int64) error
	})
	if !ok {
		return fs.ErrInvalid
	}
	return s.Truncate(name, size)
}

type fuseFs struct {
	pathfs.FileSystem
	fsys fs.FS
}

type fuseFile struct {
	nodefs.File
	path  string
	fsys  fs.FS
	fstat fs.FileInfo
}

func fixPath(name string) string {
	if name == "" {
		return "."
	}
	return name
}

func (t *fuseFs) GetAttr(name string, context *fuse.Context) (*fuse.Attr, fuse.Status) {
	name = fixPath(name)
	f, err := fs.Stat(t.fsys, name)
	if err != nil {
		return nil, fuse.ENOENT
	}

	if f.IsDir() {
		return &fuse.Attr{
			Mode: fuse.S_IFDIR | 0755,
		}, fuse.OK
	}
	return &fuse.Attr{
		Mode:  fuse.S_IFREG | 0644,
		Size:  uint64(f.Size()),
		Ctime: uint64(f.ModTime().Unix()),
		Mtime: uint64(f.ModTime().Unix()),
		Atime: uint64(f.ModTime().Unix()),
	}, fuse.OK
}

func (t *fuseFs) OpenDir(name string, context *fuse.Context) (c []fuse.DirEntry, code fuse.Status) {
	name = fixPath(name)
	files, err := fs.ReadDir(t.fsys, name)
	if err != nil {
		return nil, fuse.ENOENT
	}

	result := []fuse.DirEntry{}
	for _, f := range files {
		result = append(result, fuse.DirEntry{Name: f.Name(), Mode: fuse.S_IFREG})
	}

	return result, fuse.OK
}

func (t *fuseFs) Open(name string, flags uint32, context *fuse.Context) (file nodefs.File, code fuse.Status) {
	name = fixPath(name)
	f, err := fs.Stat(t.fsys, name)
	if err != nil {
		return nil, fuse.ENOENT
	}
	if flags&fuse.O_ANYWRITE != 0 {
		return nil, fuse.EPERM
	}

	return &fuseFile{File: nodefs.NewDefaultFile(), fstat: f, fsys: t.fsys, path: name}, fuse.OK
}

func (t *fuseFs) Create(name string, flags uint32, mode uint32, context *fuse.Context) (nodefs.File, fuse.Status) {
	name = fixPath(name)

	fsys, ok := t.fsys.(interface {
		OpenWriter(string, int) (io.WriteCloser, error)
	})
	if !ok {
		return nil, fuse.ENOSYS
	}

	// TOUCH
	ff, err := fsys.OpenWriter(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		return nil, fuse.ENOSYS
	}
	ff.Close()

	return &fuseFile{File: nodefs.NewDefaultFile(), fstat: nil, fsys: t.fsys, path: name}, fuse.OK
}

func (f *fuseFile) Read(buf []byte, off int64) (fuse.ReadResult, fuse.Status) {
	ff, err := f.fsys.Open(f.path)
	if err != nil {
		return nil, fuse.ENOSYS
	}
	defer ff.Close()

	len, err := readAt(ff, buf, off)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, fuse.ENOSYS
	}

	return fuse.ReadResultData(buf[:len]), fuse.OK
}

func (f *fuseFile) Write(data []byte, off int64) (uint32, fuse.Status) {
	fsys, ok := f.fsys.(interface {
		OpenWriter(string, int) (io.WriteCloser, error)
	})
	if !ok {
		return 0, fuse.ENOSYS
	}

	ff, err := fsys.OpenWriter(f.path, os.O_RDWR|os.O_CREATE)
	if err != nil {
		return 0, fuse.ENOSYS
	}
	defer ff.Close()

	len, err := writeAt(ff, data, off)
	if err != nil {
		return 0, fuse.ENOSYS
	}
	return uint32(len), fuse.OK
}

func (f *fuseFile) Truncate(size uint64) fuse.Status {
	err := truncate(f.fsys, f.path, int64(size))
	if err != nil {
		return fuse.ENOSYS
	}
	return fuse.OK
}

type handle struct {
	nfs    *pathfs.PathNodeFs
	server *fuse.Server
}

func (h *handle) Close() error {
	return h.server.Unmount()
}

func MountFS(mountPoint string, fsys fs.FS, opt *MountOptions) (io.Closer, error) {
	nfs := pathfs.NewPathNodeFs(&fuseFs{FileSystem: pathfs.NewDefaultFileSystem(), fsys: fsys}, nil)
	server, _, err := nodefs.MountRoot(mountPoint, nfs.Root(), nil)
	if err != nil {
		return nil, err
	}
	h := &handle{nfs: nfs, server: server}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	go func() {
		<-sig
		err := h.Close()
		if err != nil {
			fmt.Printf("Failed to unmount %s, you should umount manually: %v\n", mountPoint, err)
		}
		os.Exit(1)
	}()

	go server.Serve()
	server.WaitMount()
	return h, err
}
