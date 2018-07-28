// Package hubhub is a set of utility functions for working with the GitHub API.
//
// Copyright 2018 © Martin Tournoij
// See the bottom of this file for the full copyright.
package hubhub // import "arp242.net/hubhub"

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"sync"
	"time"
)

// GitHub credentials.
var (
	User  string
	Token string
	API   = "https://api.github.com"
)

// Request something from the GitHub API.
func Request(scan interface{}, method, url string) (*http.Response, error) {
	client := http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	//if args.header != nil {
	//	req.Header = args.header
	//}

	if User != "" && Token != "" {
		req.SetBasicAuth(User, Token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return resp, err
	}
	defer resp.Body.Close() // nolint: errcheck

	// TODO: check for non-2xx status?

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return resp, err
	}

	//fmt.Println(string(data))

	err = json.Unmarshal(data, scan)
	return resp, err
}

// RequestStat is like Request, but retry on 202 response codes.
//
// From the GitHub API docs:
//
// If the data hasn't been cached when you query a repository's statistics,
// you'll receive a 202 response; a background job is also fired to start
// compiling these statistics. Give the job a few moments to complete, and then
// submit the request again. If the job has completed, that request will receive
// a 200 response with the statistics in the response body.
func RequestStat(scan interface{}, method, url string, maxWait time.Duration) error {
	start := time.Now()
	for {
		if start.Sub(start) > maxWait {
			return errors.New("timed out")
		}

		resp, err := Request(scan, method, url)
		if err != nil {
			return err
		}

		if resp.StatusCode == 202 {
			time.Sleep(2 * time.Second)
		}
		if resp.StatusCode == 200 {
			break
		}
	}

	return nil
}

// Repository in GitHub.
type Repository struct {
	Name     string    `json:"name"`
	Language string    `json:"language"`
	PushedAt time.Time `json:"pushed_at"`
	Topics   []string  `json:"topics"`
}

type info struct {
	PublicRepos  int `json:"public_repos"`
	PrivateRepos int `json:"total_private_repos"`
}

// ListRepos lists all repositories for a user or organisation.
//
// The name is in the form of "orgs/OrganisationName" or "user/Username".
func ListRepos(name string) ([]Repository, error) {
	// Get count of repositories so we can speed parallelize the pagination.
	// Speeds up large organisations/users at the expense of slowing down
	// smaller ones.
	var i info
	_, err := Request(&i, "GET", fmt.Sprintf("%s/%s", API, name))
	if err != nil {
		return nil, err
	}

	var (
		nPages   = int(math.Ceil((float64(i.PublicRepos) + float64(i.PrivateRepos)) / 100.0))
		allRepos []Repository
		errs     []error
		lock     sync.Mutex
		wg       sync.WaitGroup
	)

	wg.Add(nPages)
	for i := 1; i <= nPages; i++ {
		go func(i int) {
			defer wg.Done()

			var repos []Repository
			_, err := Request(&repos, "GET", fmt.Sprintf("%s/%s/repos?per_page=100&page=%d", API, name, i))
			if err != nil {
				lock.Lock()
				errs = append(errs, err)
				lock.Unlock()
				return
			}
			lock.Lock()
			allRepos = append(allRepos, repos...)
			lock.Unlock()
		}(i)
	}
	wg.Wait()

	if len(errs) > 0 {
		return allRepos, fmt.Errorf("%#v", errs)
	}
	return allRepos, err
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
