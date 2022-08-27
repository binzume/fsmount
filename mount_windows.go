package fsmount

import (
	"io"
	"io/fs"
	"syscall"
	"time"

	"github.com/binzume/dkango"
)

func fileATime(fi fs.FileInfo) time.Time {
	if attr, ok := fi.Sys().(*syscall.Win32FileAttributeData); ok {
		return time.Unix(0, attr.LastAccessTime.Nanoseconds())
	}
	return fi.ModTime()
}

func fileCTime(fi fs.FileInfo) time.Time {
	if attr, ok := fi.Sys().(*syscall.Win32FileAttributeData); ok {
		return time.Unix(0, attr.CreationTime.Nanoseconds())
	}
	return fi.ModTime()
}

func MountFS(mountPoint string, fsys fs.FS, opt interface{}) (io.Closer, error) {
	mountOpt, _ := opt.(*dkango.MountOptions)
	return dkango.MountFS(mountPoint, fsys, mountOpt)
}
