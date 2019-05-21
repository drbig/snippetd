package main

import (
	"expvar"
	"flag"
	"fmt"
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
	VERSION = `0.0.2`
)

var build = `UNKNOWN` // injected via Makefile

const (
	ACCEPTED_METHOD = `POST`
	MAX_LENGTH      = 1600 // 32*50, plenty
	BUF_SIZE        = 32   // way too many for buffered messages
	STAMP_LAYOUT    = `2006-01-02 15:04:05 MST`
)

var (
	flagHost    string
	flagPort    int
	cntRequests = expvar.NewInt("_requests")
	cntPrints   = expvar.NewInt("_prints")
	cntErrors   = expvar.NewInt("_errors")
	chSnippets  = make(chan *Snippet, BUF_SIZE)
)

type Snippet struct {
	Id     int64
	Source string
	Stamp  time.Time
	Body   []byte
}

func (s *Snippet) DebugPrint() {
	fmt.Printf(`%s
--------------------------------
%s
--------------------------------
#%d @ %s
`, s.Stamp.Format(STAMP_LAYOUT), s.Body, s.Id, s.Source)
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
	go runServerPrint()
	go runServerHTTP()
	sigwait()
}

func runServerPrint() {
	log.Println("Print: Started")
	for {
		s := <-chSnippets
		log.Printf("Print: Snippet [%d] received\n", s.Id)
		s.DebugPrint()
	}
}

func runServerHTTP() {
	addr := fmt.Sprintf("%s:%d", flagHost, flagPort)
	http.HandleFunc("/print", handlePrint)
	log.Println("HTTP: Started at", addr)
	log.Fatalln(http.ListenAndServe(addr, nil))
}

func handlePrint(w http.ResponseWriter, req *http.Request) {
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
	}
	chSnippets <- &snippet
	log.Printf("HTTP: [%d] Handled\n", rid)
	fmt.Fprintf(w, "Queued as snippet %d\n", rid)
}

func dieOnErr(msg string, err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, msg+":", err)
		os.Exit(3)
	}
}

func sigwait() {
	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

	<-sig
	log.Printf("\nSignal received, stopping\n")

	return
}
