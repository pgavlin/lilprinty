package main

import (
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/pgavlin/lilprinty/internal/bitmap"
	"github.com/pgavlin/lilprinty/internal/markdown"
	"github.com/pgavlin/lilprinty/internal/printer"
)

type server struct {
	defaultStyle style
	printer      *printer.Device
}

func (s *server) handlePrint(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	isPreview := req.URL.Query().Get("preview") != ""

	contents, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Printf("error reading request body: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	device, preview := bitmap.Device(s.printer), preview{}
	if isPreview {
		device = &preview
	}

	if err = markdown.Render(device, contents, s.defaultStyle.proportionalFamily, s.defaultStyle.monospaceFamily, s.defaultStyle.headingStyles, s.defaultStyle.paragraphStyle); err != nil {
		log.Printf("error rendering content: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !isPreview {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Add("Content-Type", "image/png")
	if err = png.Encode(w, &preview); err != nil {
		log.Printf("error encoding preview result: %v", err)
	}
}

func serveFile(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		http.ServeFile(w, req, path)
	}
}

func serve(address string, defaultStyle style, w io.Writer) error {
	server := &server{
		defaultStyle: defaultStyle,
		printer:      printer.New(w),
	}
	http.HandleFunc("/print", server.handlePrint)
	http.HandleFunc("/", serveFile("./index.html"))
	http.HandleFunc("/index.css", serveFile("./index.css"))
	http.HandleFunc("/index.js", serveFile("./index.js"))
	return http.ListenAndServe(address, nil)
}
