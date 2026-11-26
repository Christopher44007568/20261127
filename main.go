package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"meshbool/internal/report"
)

func main() {
	outDir := flag.String("out", "dist", "directory for the generated report and OBJ files")
	addr := flag.String("addr", ":8080", "HTTP listen address")
	once := flag.Bool("once", false, "generate the report and exit")
	flag.Parse()

	page, err := report.Build(*outDir)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("mesh boolean: %d scenes, %d operations, %d triangles\n",
		page.Summary.Cases, page.Summary.Operations, page.Summary.Triangles)
	fmt.Printf("report: %s\n", filepath.Join(*outDir, "index.html"))
	if *once {
		return
	}

	fileServer := http.FileServer(http.Dir(*outDir))
	mux := http.NewServeMux()
	mux.Handle("/", fileServer)
	displayAddr := *addr
	if strings.HasPrefix(displayAddr, ":") {
		displayAddr = "localhost" + displayAddr
	}
	fmt.Printf("serving http://%s\n", displayAddr)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		if err == http.ErrServerClosed {
			return
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
