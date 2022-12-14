# Simple FUSE bindings for Go fs.FS
[![Build Status](https://github.com/binzume/fsmount/actions/workflows/test.yaml/badge.svg)](https://github.com/binzume/fsmount/actions)
[![Go Reference](https://pkg.go.dev/badge/github.com/binzume/fsmount.svg)](https://pkg.go.dev/github.com/binzume/fsmount)
[![license](https://img.shields.io/badge/license-MIT-4183c4.svg)](https://github.com/binzume/fsmount/blob/master/LICENSE)

This library is just a wrapper to easily mount fs.FS as a filesystem.

Dependencies:

- Windows: Dokan + [dkango](https://github.com/binzume/dkango)
- Linux: fuse + [go-fuse](https://github.com/hanwen/go-fuse)
- MacOS: macFUSE + [go-fuse](https://github.com/hanwen/go-fuse)

## Usage

### Setup FUSE

Windows:

```sh
winget install dokan-dev.Dokany
```

Linux(Ubuntu)

```sh
apt install fuse
```

### examples/simple/simple.go

```go
package main

import (
	"os"
	"github.com/binzume/fsmount"
)

func main() {
	mount, _ := fsmount.MountFS("X:", os.DirFS("."), nil)
	defer mount.Close()

	// Block forever
	select {}
}
```

### How to create a writable FS?

See examples/writable/writable.go

```
go run ./examples/writable testdir R:
```

```go
type WritableFS interface {
	fs.FS
	OpenWriter(name string, flag int) (io.WriteCloser, error)
	Truncate(name string, size int64) error
}
```

Other interfaces such as RemoveFS, MkdirFS, RenameFS... are also available.

## License

MIT
