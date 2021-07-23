package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/theoremoon/httpdump/data"
	"golang.org/x/xerrors"
)

func run() error {
	var dumpfile string
	var target string
	var errorlog string
	flag.StringVar(&dumpfile, "dumpfile", "httpdump.json", "")
	flag.StringVar(&target, "target", "", "http://localhost:1333/")
	flag.StringVar(&errorlog, "errorlog", "errorlog.json", "")
	flag.Parse()

	f, err := os.Open(dumpfile)
	if err != nil {
		return xerrors.Errorf(": %w", err)
	}
	defer f.Close()

	errf, err := os.Create(errorlog)
	if err != nil {
		return xerrors.Errorf(": %w", err)
	}
	defer errf.Close()
	errw := bufio.NewWriter(errf)

	reqcnt := 0
	errcnt := 0
	var reqres data.RequestResponse
	reader := bufio.NewReader(f)
	for {
		blob, err := reader.ReadBytes(byte('\n'))
		if err != nil {
			if xerrors.Is(err, io.EOF) {
				break
			}
			return xerrors.Errorf(": %w", err)
		}

		if err := json.Unmarshal(blob, &reqres); err != nil {
			return xerrors.Errorf(": %w", err)
		}

		// replay request
		client := &http.Client{}
		req, err := data.MakeReqeust(&reqres.Request, target)
		if err != nil {
			return xerrors.Errorf(": %w", err)
		}
		log.Printf("[Request] %s\n", req.URL.String())

		res, err := client.Do(req)
		if err != nil {
			return xerrors.Errorf(": %w", err)
		}
		defer res.Body.Close()
		reqcnt++

		// verify response
		is_valid := true
		if res.Status != reqres.Response.Status {
			log.Printf("[!Bad Status] %s is expected, but %s is got\n", reqres.Response.Status, res.Status)
			is_valid = false
		}
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return xerrors.Errorf(": %w", err)
		}
		if !bytes.Equal(body, reqres.Response.Body) {
			log.Printf("[!Bad Response] Response body does not equals to expected\n")
			is_valid = false
		}

		if !is_valid {
			errcnt++
			j, err := json.Marshal(map[string]interface{}{
				"request": reqres.Request,
				"response": data.Response{
					Status: res.Status,
					Proto:  res.Proto,
					Header: data.EncodeHeader(res.Header),
					Body:   body,
				},
				"expected": reqres.Response,
			})
			if err != nil {
				return xerrors.Errorf(": %w", err)
			}
			errw.Write(j)
			errw.WriteString("\n")
		}
	}
	log.Println("All requests have done")
	log.Printf("%d/%d errors reported\n", errcnt, reqcnt)

	return nil
}

func main() {
	if err := run(); err != nil {
		log.Printf("%+v\n", err)
	}
}
