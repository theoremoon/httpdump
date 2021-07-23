package data

import (
	"bytes"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/xerrors"
)

type Request struct {
	Method string            `json:"method"`
	URL    string            `json:"url"`
	Proto  string            `json:"proto"`
	Host   string            `json:"host"`
	Header map[string]string `json:"header"`
	Body   []byte            `json:"body"`
}

type Response struct {
	Status string            `json:"status"`
	Proto  string            `json:"proto"`
	Header map[string]string `json:"header"`
	Body   []byte            `json:"body"`
}

type RequestResponse struct {
	Request  Request  `json:"request"`
	Response Response `json:"response"`
}

func EncodeHeader(h http.Header) map[string]string {
	hmap := make(map[string]string)
	for name := range h {
		hmap[name] = h.Get(name)
	}
	return hmap
}

func MakeReqeust(r *Request, target string) (*http.Request, error) {
	u, err := url.Parse(strings.TrimRight(target, "/") + r.URL)
	if err != nil {
		return nil, xerrors.Errorf(": %w", err)
	}

	req, err := http.NewRequest(r.Method, u.String(), bytes.NewReader(r.Body))
	if err != nil {
		return nil, xerrors.Errorf(": %w", err)
	}
	for name, value := range r.Header {
		req.Header.Set(name, value)
	}
	return req, nil
}
