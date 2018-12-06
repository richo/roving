package client

import (
	"encoding/json"
	"time"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	goji "goji.io"
	"goji.io/pat"

	"github.com/richo/roving/types"
)

var suceedAfterNRequests int = 4
var requestN int

func getQueuesSometimesFail(w http.ResponseWriter, r *http.Request) {
	if requestN >= suceedAfterNRequests {
		w.WriteHeader(http.StatusOK)

		queues := make(map[string]types.InputCorpus)
		encoder := json.NewEncoder(w)
		encoder.Encode(queues)
	} else {
		// I am a teapot
		w.WriteHeader(http.StatusTeapot)
	}
	requestN++
}

func resetCounter() {
	requestN = 0
}

func TestRetriesEventualSuccess(t *testing.T) {
	mux := goji.NewMux()
	mux.HandleFunc(pat.Get("/queue"), getQueuesSometimesFail)

	resetCounter()

	serverStub := httptest.NewServer(mux)
	serverClient := RovingServerClient{
		hostport:   serverStub.URL,
		httpClient: &http.Client{},
		retryDelay: time.Duration(0) * time.Second,
		maxRetries: suceedAfterNRequests + 1,
	}

	_, err := serverClient.FetchQueues()
	assert.Empty(t, err, "Should have eventually succeeded to retrieve queue")
}

func TestRetriesExhausted(t *testing.T) {
	mux := goji.NewMux()
	mux.HandleFunc(pat.Get("/queue"), getQueuesSometimesFail)

	resetCounter()

	serverStub := httptest.NewServer(mux)
	serverClient := RovingServerClient{
		hostport:   serverStub.URL,
		httpClient: &http.Client{},
		retryDelay: time.Duration(0) * time.Second,
		maxRetries: suceedAfterNRequests - 1,
	}

	_, err := serverClient.FetchQueues()
	assert.NotEmpty(t, err, "Should have failed to retrieve queue")
}
