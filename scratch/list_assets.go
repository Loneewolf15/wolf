package main

import (
	"fmt"
	"wolf"
	"io/fs"
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
