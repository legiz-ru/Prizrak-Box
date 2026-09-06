//go:build ignore

package main

import (
	"bytes"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
)

func main() {
	names := []string{"tray-win-light", "tray-win-dark", "tray-win-light-active", "tray-win-dark-active"}
	for _, name := range names {
		path := filepath.Join("build", name+".png")
		in, err := os.ReadFile(path)
		if err != nil {
			panic(err)
		}
		img, err := png.Decode(bytes.NewReader(in))
		if err != nil {
			panic(err)
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			panic(err)
		}
		if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
			panic(err)
		}
		fmt.Printf("%s.png: %d -> %d bytes\n", name, len(in), len(buf.Bytes()))
	}
}
