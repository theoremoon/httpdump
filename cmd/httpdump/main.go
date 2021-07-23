package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/theoremoon/httpdump/data"

	"golang.org/x/xerrors"
)

func wrapReverseProxy(rp *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqBody, _ := ioutil.ReadAll(r.Body)
		ctx = context.WithValue(ctx, "request", data.Request{
			Method: r.Method,
			URL:    r.URL.String(),
			Proto:  r.Proto,
			Host:   r.Host,
			Header: data.EncodeHeader(r.Header),
			Body:   reqBody,
		})
		r = r.WithContext(ctx)

		// body差し替える
		r.Body.Close()
		r.Body = ioutil.NopCloser(bytes.NewReader(reqBody))
		rp.ServeHTTP(w, r)
	}
}

func dumpHTTP(w io.Writer) func(*http.Response) error {
	return func(res *http.Response) error {
		// dump request and response
		req := res.Request.Context().Value("request").(data.Request)
		resBody, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return xerrors.Errorf(": %w", err)
		}
		// Bodyを一度読んでしまったので差し替えておく
		res.Body.Close()
		res.Body = ioutil.NopCloser(bytes.NewReader(resBody))

		reqres := data.RequestResponse{
			Request: req,
			Response: data.Response{
				Status: res.Status,
				Proto:  res.Proto,
				Header: data.EncodeHeader(res.Header),
				Body:   resBody,
			},
		}
		jsondump, err := json.Marshal(reqres)
		if err != nil {
			return xerrors.Errorf(": %w", err)
		}
		if _, err := w.Write(jsondump); err != nil {
			return xerrors.Errorf(": %w", err)
		}
		// newline
		if _, err := w.Write([]byte("\n")); err != nil {
			return xerrors.Errorf(": %w", err)
		}
		return nil
	}
}

func run() error {
	var backendURL string
	var listenURL string
	var dumpTo string
	flag.StringVar(&backendURL, "backend", "", "http://backend:port/")
	flag.StringVar(&listenURL, "listen", "0.0.0.0:5000", "")
	flag.StringVar(&dumpTo, "out", "httpdump.json", "")

	flag.Parse()
	backend, err := url.Parse(backendURL)
	if err != nil {
		return xerrors.Errorf(": %w", err)
	}

	f, err := os.Create(dumpTo)
	if err != nil {
		return xerrors.Errorf(": %w", err)
	}
	defer f.Close()
	w := bufio.NewWriter(f)

	revProxy := httputil.NewSingleHostReverseProxy(backend)
	revProxy.ModifyResponse = dumpHTTP(w)

	srv := http.Server{
		Addr:    listenURL,
		Handler: http.HandlerFunc(wrapReverseProxy(revProxy)),
	}

	if err := srv.ListenAndServe(); err != nil {
		return xerrors.Errorf(": %w", err)
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("%v\n", err)
	}
}
