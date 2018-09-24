package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-kit/kit/endpoint"
	multierror "github.com/hashicorp/go-multierror"
)

func main() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	endpoints := map[string]endpoint.Endpoint{
		"TRAIN L": func(ctx context.Context, r interface{}) (interface{}, error) {
			if r.(bool) {
				log.Println("Ligne L a des problèmes")
			}
			return nil, nil
		},
		"TRAIN J": func(ctx context.Context, r interface{}) (interface{}, error) {
			if r.(bool) {
				log.Println("Ligne J a des problèmes")
			}
			return nil, nil
		},
	}
	ctx := context.Background()
	if err := checkTransport(ctx, endpoints); err != nil {
		log.Println(err)
	}
	for now := range ticker.C {
		_ = now
		if err := checkTransport(ctx, endpoints); err != nil {
			log.Println(err)
		}
	}
}

func checkTransport(ctx context.Context, endpoints map[string]endpoint.Endpoint) error {
	data, err := getHTTP()
	if err != nil {
		return err
	}
	html := string(data)
	var result *multierror.Error
	for k, e := range endpoints {
		okL, err := isLineOK(html, k)
		result = multierror.Append(result, err)
		if err == nil {
			_, e := e(ctx, okL)
			result = multierror.Append(result, e)
		}
	}
	return result
}

func isLineOK(html, line string) (bool, error) {
	incidents, err := incidentsOfLine(html, line)
	if err != nil {
		return false, err
	}
	return len(incidents) > 0, nil
}

func incidentsOfLine(html, line string) ([]string, error) {
	idx := strings.Index(html, fmt.Sprintf(`alt="Ligne %s"`, line))
	if idx < 0 {
		return nil, errors.New("Can't find the line " + line)
	}
	html = html[idx:]

	idx = strings.Index(html, "<td")
	if idx < 0 {
		return nil, errors.New("Can't find the trafic block for " + line)
	}
	html = html[idx:]

	idx = strings.Index(html, "</td>")
	if idx < 0 {
		return nil, errors.New("Can't find the trafic end of block for " + line)
	}
	html = html[:idx]

	var incidents []string
	for i, href := range strings.Split(html, `href="`) {
		if i == 0 {
			continue
		}
		idx = strings.Index(href, `"`)
		if idx > 0 {
			href = href[:idx]
		}
		incidents = append(incidents, strings.TrimSpace(href))
	}
	log.Println(line, incidents)
	return incidents, nil
}

func getHTTP() ([]byte, error) {
	res, err := http.Get("https://transilien.mobi/trafic")
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	content, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return content, nil
}
