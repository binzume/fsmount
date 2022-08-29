package fsmount

import (
	"io"
	"io/fs"
)

type MountOptions struct {
	FuseOption interface{}
}

type OpenWriterFS interface {
	fs.FS
	OpenWriter(name string, flag int) (io.WriteCloser, error)
}

type RemoveFS interface {
	fs.FS
	Remove(name string) error
}

type RenameFS interface {
	fs.FS
	Rename(name string, newName string) error
}

type MkdirFS interface {
	fs.FS
	Mkdir(name string, mode fs.FileMode) error
}

type OpenDirFS interface {
	fs.FS
	OpenDir(name string) (fs.ReadDirFile, error)
}

type TruncateFS interface {
	fs.FS
	Truncate(name string, size int64) error
}
