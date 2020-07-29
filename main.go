package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/tarm/serial"

	"github.com/pgavlin/lilprinty/internal/printer"
	"github.com/pgavlin/lilprinty/internal/renderer"
)

func main() {
	var port, filePath, serveAddress, stylePath string
	flag.StringVar(&port, "port", "", "the serial port to use for the printer")
	flag.StringVar(&stylePath, "style", "", "the path to the stylesheet, if any")
	flag.StringVar(&filePath, "file", "", "the path to the file to print, if any")
	flag.StringVar(&serveAddress, "serve", "", "the address to serve on, if any")
	flag.Parse()

	if filePath != "" && serveAddress != "" {
		fmt.Fprintf(os.Stderr, "only one of -file and -serve may be specified")
		os.Exit(-1)
	}

	var w io.Writer
	if port != "" {
		s, err := serial.OpenPort(&serial.Config{Name: port, Baud: 9600})
		if err != nil {
			log.Fatalf("error opening '%v': %v", port, err)
		}
		w = s
	} else {
		w = os.Stdout
	}

	style := defaultStyle
	if stylePath != "" {
		s, err := loadStylesheet(stylePath)
		if err != nil {
			log.Fatalf("error loading style sheet: %v", err)
		}
		style = s
	}

	if filePath != "" {
		bytes, err := ioutil.ReadFile(filePath)
		if err != nil {
			log.Fatalf("error reading '%v': %v", filePath, err)
		}
		if err = renderer.RenderMarkdown(printer.New(w), bytes, style.proportionalFamily, style.monospaceFamily, style.headingStyles, style.paragraphStyle); err != nil {
			log.Fatalf("error rendering document: %v", err)
		}
	} else {
		if err := serve(serveAddress, style, w); err != nil {
			log.Fatalf("serve error: %v", err)
		}
	}
}
