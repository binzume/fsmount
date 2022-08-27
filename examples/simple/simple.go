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
