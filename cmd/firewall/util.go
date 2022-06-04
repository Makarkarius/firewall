package main

import (
	"bytes"
	"log"
	"net/http"
	"regexp"
	"strings"
)

type ruleList struct {
	Endpoint string `yaml:"endpoint"`
	// both
	ForbiddenHeaders []string `yaml:"forbidden_headers"`
	RequiredHeaders  []string `yaml:"required_headers"`
	// request
	ForbiddenUserAgents []string `yaml:"forbidden_user_agents"`
	ForbiddenRequestRe  []string `yaml:"forbidden_request_re"`
	MaxReqLen           int      `yaml:"max_request_length_bytes"`
	// response
	ForbiddenResponseCodes []int    `yaml:"forbidden_response_codes,flow"`
	ForbiddenResponseRe    []string `yaml:"forbidden_response_re"`
	MaxRespLen             int      `yaml:"max_response_length_bytes"`
}

type cfg struct {
	Rules []ruleList `yaml:"rules,flow"`
}

func checkHeaders(header *http.Header, forbidden *[]string, required *[]string) bool {
	for _, fh := range *forbidden {
		tokens := strings.Split(fh, ": ")
		key := tokens[0]
		val := tokens[1]
		for _, head := range header.Values(key) {
			if head == val {
				return false
			}
		}
	}
	for _, fh := range *required {
		if len(header.Values(fh)) == 0 {
			return false
		}
	}
	return true
}

func checkRequest(rl *ruleList, r *http.Request) (bool, error) {
	if rl == nil {
		return true, nil
	}

	if !checkHeaders(&r.Header, &rl.ForbiddenHeaders, &rl.RequiredHeaders) {
		return false, nil
	}

	for _, userAgent := range rl.ForbiddenUserAgents {
		matched, err := regexp.Match(userAgent, []byte(r.UserAgent()))
		if err != nil {
			log.Fatalf("Invalid expression\nexpression: %s\n%s", userAgent, err.Error())
		}
		if matched {
			return false, nil
		}
	}

	body := new(bytes.Buffer)
	if _, e := body.ReadFrom(r.Body); e != nil {
		return false, e
	}
	for _, requestRe := range rl.ForbiddenRequestRe {
		matched, err := regexp.Match(requestRe, body.Bytes())
		if err != nil {
			log.Fatalf("Invalid expression\nexpression: %s\n%s", requestRe, err.Error())
		}
		if matched {
			return false, nil
		}
	}
	if len(body.String()) > rl.MaxReqLen {
		return false, nil
	}
	return true, nil
}

func checkResponse(rl *ruleList, r *http.Response) (bool, error) {
	if rl == nil {
		return true, nil
	}

	if !checkHeaders(&r.Header, &rl.ForbiddenHeaders, &rl.RequiredHeaders) {
		return false, nil
	}

	for _, code := range rl.ForbiddenResponseCodes {
		if r.StatusCode == code {
			return false, nil
		}
	}

	body := new(bytes.Buffer)
	if _, e := body.ReadFrom(r.Body); e != nil {
		return false, e
	}
	for _, responseRe := range rl.ForbiddenResponseRe {
		matched, err := regexp.Match(responseRe, body.Bytes())
		if err != nil {
			log.Fatalf("Invalid expression\nexpression: %s\n%s", responseRe, err.Error())
		}
		if matched {
			return false, nil
		}
	}
	if len(body.String()) > rl.MaxRespLen {
		return false, nil
	}
	return true, nil
}
