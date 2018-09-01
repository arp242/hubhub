// Copyright 2018 © Martin Tournoij
// See the bottom of this file for the full copyright.

// Package hubhub is a set of utility functions for working with the GitHub API.
//
// It's not a "client library"; but just a few convenience functions that I
// found myself re-creating a bunch of times in different programs.
package hubhub // import "arp242.net/hubhub"

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"
)

// Global configuration.
var (
	User      string                     // GitHub username.
	Token     string                     // GitHub access token or password.
	API       = "https://api.github.com" // API base URL.
	DebugURL  = false                    // Show URLs as they're requested.
	DebugBody = false                    // Show body of requests.
	MaxWait   = 30 * time.Second         // Max time to wait on 202 Accepted responses.
)

// NotOKError is used when the status code is not 200 OK.
type NotOKError struct {
	Method, URL string
	Status      string
	StatusCode  int
}

func (e NotOKError) Error() string {
	return fmt.Sprintf("code %s for %s %s", e.Status, e.Method, e.URL)
}

// ErrWait is used when we've waited longer than MaxWait for 202 Accepted.
var ErrWait = errors.New("waited longer than MaxWait and still getting 202 Accepted")

var client = http.Client{Timeout: 10 * time.Second}

// Request something from the GitHub API.
//
// The response body will be unmarshaled in to scan unless the response code is
// 204 (No Content).
//
// If the response code is 202 (Accepted) the HTTP request will be retried every
// two seconds until it returns a 200 OK with data. The ErrWait error will be
// returned if this takes longer than MaxWait.
//
// A response code higher than 399 will return a NotOKError error, but won't
// affect the behaviour of this function.
//
// The Body on the returned *http.Response is closed.
//
// This will use the global User and Token, which must be set.
func Request(scan interface{}, method, url string) (*http.Response, error) {
	start := time.Now()

	if User == "" || Token == "" {
		panic("hubhub: must set User and Token")
	}

	if !strings.HasPrefix(url, "https://") {
		url = API + url
	}

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", fmt.Sprintf(
		"Go-http-client/1.1; User=%s; client=hubhub", User))

	if User != "" && Token != "" {
		req.SetBasicAuth(User, Token)
	}

doreq:
	if DebugURL {
		fmt.Printf("%v %v\n", method, url)
	}
	resp, err := client.Do(req)
	if err != nil {
		return resp, err
	}
	defer resp.Body.Close() // nolint: errcheck

	// 202 Accepted: re-try the request after a short delay.
	if resp.StatusCode == http.StatusAccepted {
		if start.Sub(start) > MaxWait {
			return resp, ErrWait
		}
		time.Sleep(2 * time.Second)
		goto doreq
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return resp, err
	}

	if DebugBody {
		fmt.Println(string(data))
	}

	// Some endpoints return 204 when there is no content (e.g. getting
	// information about a repo without any code).
	if resp.StatusCode != http.StatusNoContent {
		err = json.Unmarshal(data, scan)
	}

	if resp.StatusCode >= 400 {
		// Intentionally override the JSON status error; chances are this is the
		// root cause.
		err = NotOKError{
			Status:     resp.Status,
			StatusCode: resp.StatusCode,
			Method:     method,
			URL:        url,
		}
	}

	return resp, err
}

// Paginate an index request.
//
// if nPages is higher than zero it will get exactly that number of pages in
// parallel. If it is zero it will get every page in serial until the last
// page.
//
// TODO: Could be prettier.
// TODO: Allow specifying per_page
// TODO: Method is probably superfluous as all requests with pagination are GET?
func Paginate(scan interface{}, method, url string, nPages int) error {
	t := reflect.TypeOf(scan)
	if t.Kind() != reflect.Ptr {
		panic("hubhub: not a pointer")
	}
	t = t.Elem()
	if t.Kind() != reflect.Slice {
		panic("hubhub: not a slice")
	}

	var (
		slice = reflect.Indirect(reflect.ValueOf(scan))
		errs  []error
		lock  sync.Mutex
		wg    sync.WaitGroup
	)

	getPage := func(i int) bool {
		if nPages > 0 {
			defer wg.Done()
		}

		s := reflect.New(t).Interface()
		_, err := Request(&s, method, fmt.Sprintf("%s?page=%d", url, i)) // TODO: better URL parsing
		if err != nil {
			lock.Lock()
			errs = append(errs, err)
			lock.Unlock()

			if nPages == 0 {
				return true
			}
			return false
		}

		v := reflect.ValueOf(s).Elem()
		if v.Len() == 0 {
			return true
		}

		lock.Lock()
		slice.Set(reflect.AppendSlice(slice, v))
		lock.Unlock()
		return false
	}

	if nPages > 0 {
		wg.Add(nPages)
	}

	l := nPages
	if l == 0 {
		l = 999
	}
	for i := 1; i <= l; i++ {
		if nPages == 0 {
			lastPage := getPage(i)
			if lastPage {
				break
			}
		} else {
			go getPage(i)
		}
	}
	if nPages > 0 {
		wg.Wait()
	}

	if len(errs) > 0 {
		return fmt.Errorf("%s", errs)
	}
	return nil
}

// Copyright 2018 © Martin Tournoij
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
