package main

import (
	"fmt"
	"io/fs"
	"github.com/wolflang/wolf"
)

func main() {
	fs.WalkDir(wolf.Assets, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			fmt.Println(path)
		}
		return nil
	})
}
