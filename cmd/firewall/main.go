//go:build !solution

package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type myTransport struct {
	config      cfg
	serviceAddr string
}

var client http.Client
var clientT http.Client

func forbiddenResponse(r *http.Request) *http.Response {
	body := "Forbidden"
	response := &http.Response{
		Status:        "403 Forbidden",
		StatusCode:    403,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Body:          ioutil.NopCloser(bytes.NewBufferString(body)),
		ContentLength: int64(len(body)),
		Request:       r,
		Header:        make(http.Header, 0),
	}
	return response
}

func (transport *myTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	var rl *ruleList

	if transport.config.Rules != nil {
		for _, rule := range transport.config.Rules {
			if rule.Endpoint == r.URL.Path {
				rl = &rule
				break
			}
		}
	}

	check, e := checkRequest(rl, r)
	if e != nil {
		return nil, e
	}
	if !check {
		return forbiddenResponse(r), nil
	}

	newRequest, e := http.NewRequest(r.Method, transport.serviceAddr, r.Body)
	newRequest.Header = r.Header
	if e != nil {
		return nil, e
	}
	response, e := clientT.Do(newRequest)
	if e != nil {
		return nil, e
	}

	if transport.config.Rules != nil {
		for _, rule := range transport.config.Rules {
			loc, e := response.Location()
			if e != nil {
				rl = nil
				break
			}
			if rule.Endpoint == loc.Path {
				rl = &rule
				break
			}
		}
	}
	check, e = checkResponse(rl, response)
	if e != nil {
		return nil, e
	}
	if !check {
		return forbiddenResponse(r), nil
	}
	return response, nil
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	response, e := client.Transport.RoundTrip(r)
	if e != nil {
		w.WriteHeader(http.StatusForbidden)
		if _, e = w.Write([]byte("Forbidden")); e != nil {
			return
		}
		return
	}
	body := new(bytes.Buffer)
	if _, e := body.ReadFrom(response.Body); e != nil {
		return
	}
	w.WriteHeader(response.StatusCode)
	if _, e := w.Write(body.Bytes()); e != nil {
		return
	}
	fmt.Println(response)
}

func main() {
	var (
		serviceAddr *string
		address     *string
		configPath  *string
		config      cfg
	)
	serviceAddr = flag.String("service-addr", "http://localhost:8080", "service address")
	address = flag.String("addr", "localhost:8081", "address")
	configPath = flag.String("conf", "./firewall/configs/example.yaml", "configuration path")
	flag.Parse()

	configData, e := ioutil.ReadFile(*configPath)
	if e != nil {
		log.Fatalf("\nCannot opend config file\ngiven path: %s,\n%s", *configPath, e.Error())
	}

	e = yaml.Unmarshal(configData, &config)
	if e != nil {
		log.Fatalf("\nCannot parse config\ngiven path: %s,\n%s", *configPath, e.Error())
	}

	client = http.Client{
		Transport: &myTransport{
			config:      config,
			serviceAddr: *serviceAddr,
		},
		Timeout: 5 * time.Second,
	}
	clientT = http.Client{
		Timeout: 5 * time.Second,
	}

	router := mux.NewRouter()
	router.HandleFunc("/{path}", handleRequest)
	if err := http.ListenAndServe(*address, router); err != nil {
		return
	}
}
