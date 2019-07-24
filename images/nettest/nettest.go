// A tiny web server for checking networking connectivity.
//
// Will dial out to, and expect to hear from, every pod that is a member of
// the service passed in the flag -service.
//
// Will serve a webserver on given -port.
//
// Visit /read to see the current state, or /quit to shut down.
//
// Visit /status to see pass/running/fail determination. (literally, it will
// return one of those words.)
//
// /write is used by other network test pods to register connectivity.

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/spf13/cobra"
)

var (
	port          int
	peerCount     int
	service       string
	namespace     string
	delayShutdown int
)

// CmdNettest is used by agnhost Cobra.
var CmdNettest = &cobra.Command{
	Use:   "nettest",
	Short: "Starts a tiny web server for checking networking connectivity",
	Long: `Starts a web server for checking networking connectivity on the given "--port".

The web server will have the following endpoints:

- "/read": to see the current state, or "/quit" to shut down.

- "/write": is used by other network test pods to register connectivity.`,
	Args: cobra.MaximumNArgs(0),
	Run:  runServer,
}

func init() {
	rootCmd.AddCommand(CmdNettest)
	CmdNettest.Flags().IntVar(&port, "port", 8080, "Port number to serve at.")
}

// State tracks the internal state of our little http server.
// It's returned verbatim over the /read endpoint.
type State struct {
	// Hostname is set once and never changed-- it's always safe to read.
	Hostname string

	// The below fields require that lock is held before reading or writing.
	Sent     map[string]int
	Received map[string]int
	Errors   []string
	Log      []string

	lock sync.Mutex
}

// serveRead writes our json encoded state
func (s *State) serveRead(w http.ResponseWriter, r *http.Request) {
	log.Print("serveRead")
	s.lock.Lock()
	defer s.lock.Unlock()
	w.WriteHeader(http.StatusOK)
	b, err := json.MarshalIndent(s, "", "\t")
	s.appendErr(err)
	_, err = w.Write(b)
	s.appendErr(err)
}

// WritePost is the format that (json encoded) requests to the /write handler should take.
type WritePost struct {
	Source string
	Dest   string
}

// WriteResp is returned by /write
type WriteResp struct {
	Hostname string
}

// serveWrite records the contact in our state.
func (s *State) serveWrite(w http.ResponseWriter, r *http.Request) {
	log.Print("serveWrite")
	defer r.Body.Close()
	s.lock.Lock()
	defer s.lock.Unlock()
	w.WriteHeader(http.StatusOK)
	var wp WritePost
	s.appendErr(json.NewDecoder(r.Body).Decode(&wp))
	if wp.Source == "" {
		s.appendErr(fmt.Errorf("%v: Got request with no source", s.Hostname))
	} else {
		if s.Received == nil {
			s.Received = map[string]int{}
		}
		s.Received[wp.Source]++
	}
	s.appendErr(json.NewEncoder(w).Encode(&WriteResp{Hostname: s.Hostname}))
}

// appendErr adds err to the list, if err is not nil. s must be locked.
func (s *State) appendErr(err error) {
	if err != nil {
		s.Errors = append(s.Errors, err.Error())
	}
}

var (
	// Our one and only state object
	state State
)

func runServer(cmd *cobra.Command, args []string) {
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("Error getting hostname: %v", err)
	}

	state := State{
		Hostname: hostname,
	}

	http.HandleFunc("/quit", func(w http.ResponseWriter, r *http.Request) {
		os.Exit(0)
	})

	http.HandleFunc("/read", state.serveRead)
	http.HandleFunc("/write", state.serveWrite)
	log.Print("Running server")

	go log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))

	select {}
}
