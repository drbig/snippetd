package main

import (
	"bytes"
	"expvar"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const (
	VERSION = `0.3.3`
)

var build = `UNKNOWN` // injected via Makefile

const (
	ACCEPTED_METHOD = `POST`
	KEY_RAW         = `raw`
	MAX_LENGTH      = 16000 // accept 16 kB, because images
	BUF_SIZE        = 32    // way too many for buffered messages
	STAMP_LAYOUT    = `2006-01-02 15:04:05 MST`
)

const (
	SMALL_START   = "\x1b\x4d\x01"
	SMALL_END     = "\x1b\x4d\x00"
	ALIGN_CENTER  = "\x1b\x61\x01"
	ALIGN_LEFT    = "\x1b\x61\x00"
	ALIGN_RIGHT   = "\x1b\x61\x02"
	RESET_PRINTER = "\x1b\x40"
)

var (
	flagHost    string
	flagPort    int
	devPath     string
	cntRequests = expvar.NewInt("_requests")
	cntPrints   = expvar.NewInt("_prints")
	cntErrors   = expvar.NewInt("_errors")
	chSnippets  = make(chan *Snippet, BUF_SIZE)
	outBuf      bytes.Buffer
	outW        = io.Writer(&outBuf)
)

type Snippet struct {
	Id     int64
	Source string
	Stamp  time.Time
	Body   []byte
	Raw    bool
}

func (s *Snippet) DebugPrint() {
	fmt.Printf(`%s
--------------------------------
%s
--------------------------------
#%d @ %s
`, s.Stamp.Format(STAMP_LAYOUT), s.Body, s.Id, s.Source)
}

func (s *Snippet) ESCPrint(w io.Writer) {
	fmt.Fprintf(w, `%s%s%s%s%s
%s--------------------------------
%s
--------------------------------
%s%s#%d @ %s%s
%s


`,
		RESET_PRINTER, ALIGN_CENTER, SMALL_START, s.Stamp.Format(STAMP_LAYOUT), SMALL_END,
		ALIGN_LEFT,
		s.Body,
		ALIGN_RIGHT, SMALL_START, s.Id, s.Source, SMALL_END,
		ALIGN_LEFT)
}

func (s *Snippet) ESCPrintRaw(w io.Writer) {
	fmt.Fprintf(w, "%s%s\n", RESET_PRINTER, s.Body)
}

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: %s (options...) <device_path>
snippetd v%s by Piotr S. Staszewski, see LICENSE.txt
binary build by %s

Options:
`, os.Args[0], VERSION, build)
		flag.PrintDefaults()
	}
	flag.StringVar(&flagHost, "h", "127.0.0.1", "address to bind the HTTP server to")
	flag.IntVar(&flagPort, "p", 9999, "port to bind the HTTP server to")
}

func main() {
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}
	devPath = flag.Arg(0)
	if _, err := os.Stat(devPath); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "Device path doesn't exist")
		os.Exit(2)
	}
	log.Printf("Starting v%s...\n", VERSION)
	go runServerPrint()
	go runServerHTTP()
	sigwait()
}

func runServerPrint() {
	log.Println("Print: Started at", devPath)
	for {
		s := <-chSnippets
		t0 := time.Now()
		log.Printf("Print: Snippet [%d] received\n", s.Id)
		fd, err := syscall.Open(devPath, os.O_APPEND|os.O_WRONLY, 0222)
		if err != nil {
			log.Printf("Print: [%d] Error opening printer: %s\n", s.Id, err)
			cntErrors.Add(1)
			return
		}
		log.Printf("Print: [%d] Printing...\n", s.Id)
		outBuf.Reset()
		if s.Raw {
			s.ESCPrintRaw(outW)
		} else {
			s.ESCPrint(outW)
		}
		if _, err := syscall.Write(fd, outBuf.Bytes()); err != nil {
			log.Printf("Print: [%d] Error writing: %s\n", s.Id, err)
			cntErrors.Add(1)
		}
		if err := syscall.Close(fd); err != nil {
			log.Fatalf("Print: [%d] Error closing printer: %s\n", s.Id, err)
		}
		t1 := time.Now()
		log.Printf("Print: [%d] Finished in %v\n", s.Id, t1.Sub(t0))
		cntPrints.Add(1)
	}
}

func runServerHTTP() {
	addr := fmt.Sprintf("%s:%d", flagHost, flagPort)
	http.HandleFunc("/print", handlePrint)
	log.Println("HTTP: Started at", addr)
	log.Fatalln(http.ListenAndServe(addr, nil))
}

func handlePrint(w http.ResponseWriter, req *http.Request) {
	t0 := time.Now()
	cntRequests.Add(1)
	rid := cntRequests.Value()
	log.Printf("HTTP: Print request [%d] %s @ %s %d\n", rid, req.Method, req.RemoteAddr, req.ContentLength)
	if req.Method != ACCEPTED_METHOD {
		log.Printf("HTTP: [%d] Wrong method\n", rid)
		cntErrors.Add(1)
		http.Error(w, "Method not supported", 400)
		return
	}
	if !((req.ContentLength > 0) && (req.ContentLength <= MAX_LENGTH)) {
		log.Printf("HTTP: [%d] Length not acceptable\n", rid)
		cntErrors.Add(1)
		http.Error(w, "Length not acceptable", 400)
		return
	}
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Printf("HTTP: [%d] Error reading body: %s\n", rid, err)
		cntErrors.Add(1)
		http.Error(w, "Problem reading body", 500)
		return
	}
	snippet := Snippet{
		Id:     rid,
		Source: req.RemoteAddr[:strings.IndexByte(req.RemoteAddr, ':')],
		Stamp:  time.Now(),
		Body:   body,
		Raw:    req.FormValue(KEY_RAW) != "",
	}
	chSnippets <- &snippet
	t1 := time.Now()
	log.Printf("HTTP: [%d] Finished in %v\n", rid, t1.Sub(t0))
	fmt.Fprintf(w, "Queued as snippet %d\n", rid)
}

func sigwait() {
	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

	<-sig
	log.Printf("\nSignal received, stopping\n")

	return
}
