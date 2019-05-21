package main

import (
	"expvar"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

const (
	VERSION = `0.0.1`
)

var build = `UNKNOWN` // injected via Makefile

const (
	ACCEPTED_METHOD = `POST`
	MAX_LENGTH      = 1600 // 32*50, plenty
)

var (
	flagHost    string
	flagPort    int
	cntRequests = expvar.NewInt("requests")
	cntPrints   = expvar.NewInt("prints")
	cntErrors   = expvar.NewInt("errors")
)

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
	sigwait()
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
		cnt.Errors.Add(1)
		http.Error(w, "Length not acceptable", 400)
		return
	}
}

func runServerHTTP() {
	addr := fmt.Sprintf("%s:%d", *flagHost, *flagPort)
	http.HandleFunc("/print", handlePrint)
	log.Println("HTTP: Started at", addr)
	log.Fatalln(http.ListenAndServe(addr, nil))
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
	log.Println("Signal received, stopping")

	return
}
