package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/richo/roving/types"
)

var dialerTimeout time.Duration = 5 * time.Second
var tlsHandshakeTimeout time.Duration = 5 * time.Second
var httpRequestTimeout time.Duration = 10 * time.Second
var retryDelay time.Duration = 5 * time.Second
var maxRetries int = 3

// RovingServerClient is a wrapper around the roving server API.
// It is used for communicating between client and server. The
// roving client uses it for operations like uploading its state
// and downloading the cluster's queues.
type RovingServerClient struct {
	hostport   string
	httpClient *http.Client
	retryDelay time.Duration
	maxRetries int
}

// NewRovingServerClient builds a RovingServerClient that points
// at the given hostport. It uses sane HTTP Client defaults,
// specified at the top of this file (server_client.go).
func NewRovingServerClient(hostport string) *RovingServerClient {
	// We don't really care about speed of sending and receiving data
	// from the server, so we have very generous timeout settings.
	httpTransport := &http.Transport{
		Dial: (&net.Dialer{
			Timeout: dialerTimeout,
		}).Dial,
		TLSHandshakeTimeout: tlsHandshakeTimeout,
	}
	httpClient := &http.Client{
		Timeout:   httpRequestTimeout,
		Transport: httpTransport,
	}

	return &RovingServerClient{
		hostport:   hostport,
		httpClient: httpClient,
		retryDelay: retryDelay,
		maxRetries: maxRetries,
	}
}

// FetchFuzzerConfig fetches the target's metadata, including
// whether the client should download the target and what command
// it should use to run it.
func (s *RovingServerClient) FetchFuzzerConfig() (*types.FuzzerConfig, error) {
	resp, err := s.makeRequest("GET", "config", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	config := &types.FuzzerConfig{}

	encoder := json.NewDecoder(resp.Body)
	encoder.Decode(config)

	return config, nil
}

// FetchTargetBinary fetches the target binary, if appropriate
func (s *RovingServerClient) FetchTargetBinary(file string) error {
	return s.fetchToFile("target/binary", file)
}

// FetchInputs fetches the inputs that AFL uses to bootstrap fuzzing.
func (s *RovingServerClient) FetchInputs() (*types.InputCorpus, error) {
	resp, err := s.makeRequest("GET", "inputs", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	inps := &types.InputCorpus{}

	encoder := json.NewDecoder(resp.Body)
	encoder.Decode(&inps)

	return inps, nil
}

// FetchDict fetches the dict of key tokens, if appropriate
func (s *RovingServerClient) FetchDict() ([]byte, error) {
	resp, err := s.makeRequest("GET", "dict", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return bodyBytes, nil
}

// FetchQueues fetches all of the queues in the roving cluster. It
// returns them as a map from fuzzerId => *InputCorpus.
func (s *RovingServerClient) FetchQueues() (*map[string]*types.InputCorpus, error) {
	resp, err := s.makeRequest("GET", "queue", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var queue map[string]*types.InputCorpus
	encoder := json.NewDecoder(resp.Body)
	encoder.Decode(&queue)

	return &queue, nil
}

// UploadState uploads the given State
func (s *RovingServerClient) UploadState(state types.State) error {
	stateJson, err := json.Marshal(state)
	if err != nil {
		return err
	}

	resp, err := s.makeRequest("POST", "state", bytes.NewReader(stateJson))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// makeRequest handles making requests to the roving server. It has some
// rudimentary retry logic that copes with transient failures.
func (s *RovingServerClient) makeRequest(method string, path string, body io.Reader) (*http.Response, error) {
	resource := fmt.Sprintf("%s/%s", s.hostport, path)
	log.Printf("Making RovingServerClient request resource=%v method=%v", resource, method)

	req, err := http.NewRequest(method, resource, body)
	if err != nil {
		return nil, err
	}

	nTries := 0
	for {
		resp, err := s.httpClient.Do(req)

		var statusCode int
		if err == nil {
			if resp.StatusCode == http.StatusOK {
				log.Printf("Suceeded RovingServerClient request status_code=%d n_tries=%d", resp.StatusCode, nTries)
				return resp, nil
			} else {
				statusCode = resp.StatusCode
			}
		}
		log.Printf("Failed RovingServerClient request status_code=%d err=%v n_tries=%d", statusCode, err, nTries)

		nTries++
		if nTries >= s.maxRetries {
			return nil, errors.New("Ran out of retries.")
		}
		time.Sleep(s.retryDelay)
	}
}

// fetchToFile retrieves a resource from the server and writes it
// to a file.
func (s *RovingServerClient) fetchToFile(resource, file string) error {
	resp, err := s.makeRequest("GET", resource, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		return err
	}

	io.Copy(f, resp.Body)
	f.Close()

	return nil
}
