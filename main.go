package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

func main() {
	out := flag.String("out", "artifacts", "directory for rendered PNG files")
	flag.Parse()

	if err := os.MkdirAll(*out, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	scenes := []struct {
		name string
		fn   func(string) error
	}{
		{"prism.png", RenderPrism},
		{"sphere.png", RenderSphere},
		{"plane.png", RenderPlate},
	}

	for _, scene := range scenes {
		start := time.Now()
		path := filepath.Join(*out, scene.name)
		if err := scene.fn(path); err != nil {
			fmt.Fprintf(os.Stderr, "render %s: %v\n", scene.name, err)
			os.Exit(1)
		}
		fmt.Printf("rendered %s in %s\n", path, time.Since(start).Round(time.Millisecond))
	}

	planePath := filepath.Join(*out, "plane.png")
	platePath := filepath.Join(*out, "plate.png")
	if err := copyFile(planePath, platePath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
