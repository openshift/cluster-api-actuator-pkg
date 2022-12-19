package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"strings"
)

const (
	awsTerminationPattern      = "/latest/meta-data/spot/termination-time"
	awsTokenPattern            = "/latest/api/token"
	gcpTerminationPattern      = "/computeMetadata/v1/instance/preempted"
	azureTerminationPattern    = "/metadata/scheduledevents"
	azureTerminationAPIVersion = "2019-08-01"
)

func main() {
	provider := flag.String("provider", "", "Cloud Provider metadata service to mock (One of AWS, Azure or GCP)")
	listenAddr := flag.String("listen-addr", "0.0.0.0:80", "Address on which metadata mock service should listen")
	flag.Parse()

	var handler http.Handler

	switch {
	case strings.EqualFold(*provider, "aws"):
		handler = awsMetadataMockHandler()
	case strings.EqualFold(*provider, "azure"):
		handler = azureMetadataMockHandler()
	case strings.EqualFold(*provider, "gcp"):
		handler = gcpMetadataMockHandler()
	default:
		log.Fatal("--provider must be one of: aws, azure or gcp")
	}

	log.Printf("Starting mock metadata service for provider: %s", *provider)

	if err := http.ListenAndServe(*listenAddr, logRequests(handler)); err != nil {
		log.Fatal(err)
	}
}

// AWS instances expect an OK response to indicate that the instance has been scheduled for termination.
func awsMetadataMockHandler() http.Handler {
	mux := http.NewServeMux()

	const (
		authTokenValue = "FOO"
		tokenHeader    = "X-aws-ec2-metadata-token"
		tokenTTLHeader = "X-aws-ec2-metadata-token-ttl-seconds"
	)

	mux.HandleFunc(awsTerminationPattern, func(rw http.ResponseWriter, req *http.Request) {
		if req.Header.Get(tokenHeader) != authTokenValue {
			rw.WriteHeader(http.StatusUnauthorized)
			return
		}
		rw.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc(awsTokenPattern, func(rw http.ResponseWriter, req *http.Request) {
		if req.Header.Get(tokenTTLHeader) == "" {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		rw.Header().Add(tokenTTLHeader, "21600")
		rw.WriteHeader(http.StatusOK)
		if _, err := rw.Write([]byte(authTokenValue)); err != nil {
			log.Fatal(err)
		}
	})

	return mux
}

// Azure instances expect an OK response with a Json body containing a schedule peremption event to
// indicate that the instance has been scheduled for termination.
// Requires the header "Metadata: true".
// Requires a query with the api version: eg `?api-version=2019-08-01`.
func azureMetadataMockHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc(azureTerminationPattern, func(rw http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Metadata") != "true" {
			// Require the "Metadata" header to be correct
			rw.WriteHeader(http.StatusBadRequest)

			if _, err := rw.Write([]byte("400 Bad Request")); err != nil {
				log.Fatal(err)
			}

			return
		}

		if req.URL.Query().Get("api-version") != azureTerminationAPIVersion {
			// Require the "api-version" query string to be correct
			rw.WriteHeader(http.StatusBadRequest)

			if _, err := rw.Write([]byte("400 Bad Request")); err != nil {
				log.Fatal(err)
			}

			return
		}

		events := azureScheduledEvents{
			Events: []azureEvent{
				{
					EventType: azurePreemptEventType,
				},
			},
		}
		data, err := json.Marshal(events)
		if err != nil {
			rw.WriteHeader(500)

			if _, err := rw.Write([]byte("500 Internal Server Error")); err != nil {
				log.Fatal(err)
			}

			return
		}

		if _, err := rw.Write(data); err != nil {
			log.Fatal(err)
		}
	})

	return mux
}

const azurePreemptEventType = "Preempt"

// azureScheduledEvents represents metadata response, more detailed info can be found here:
// https://docs.microsoft.com/en-us/azure/virtual-machines/linux/scheduled-events#use-the-api
type azureScheduledEvents struct {
	Events []azureEvent `json:"Events"`
}

type azureEvent struct {
	EventType string `json:"EventType"`
}

// GCP instances expect an OK response with the body TRUE to indicate that the instance has been
// scheduled for termination. Requires the header "Metadata-Flavor: Google".
func gcpMetadataMockHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc(gcpTerminationPattern, func(rw http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Metadata-Flavor") != "Google" {
			// Require the "Metadata-Flavor" header to be correct.
			rw.WriteHeader(http.StatusBadRequest)

			if _, err := rw.Write([]byte("400 Bad Request")); err != nil {
				log.Fatal(err)
			}

			return
		}

		if _, err := rw.Write([]byte("TRUE")); err != nil {
			log.Fatal(err)
		}
	})

	return mux
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		w := wrapResponseWriter(rw)
		next.ServeHTTP(w, req)

		log.Printf("Request: status=%d path=%q method=%q request-headers=%v", w.Status(), req.URL.EscapedPath(), req.Method, req.Header)
	})
}

// responseWriter is a minimal wrapper for http.ResponseWriter that allows the
// written HTTP status code to be captured for logging.
type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func wrapResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w}
}

func (rw *responseWriter) Status() int {
	return rw.status
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}

	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
	rw.wroteHeader = true
}

func (rw *responseWriter) Write(data []byte) (int, error) {
	if !rw.wroteHeader {
		rw.status = http.StatusOK
		rw.wroteHeader = true
	}

	return rw.ResponseWriter.Write(data)
}
