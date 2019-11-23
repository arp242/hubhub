package hubhub

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRequest(t *testing.T) {
	n := 0
	tests := []struct {
		name    string
		handler http.HandlerFunc
		want    string
		wantErr string
	}{
		{
			"regular",
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintln(w, `{"x": "y"}`)
			},
			"y",
			"",
		},
		{
			"no content",
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			},
			"",
			"",
		},
		{
			"accepted once",
			func(w http.ResponseWriter, r *http.Request) {
				if n == 0 {
					n++
					w.WriteHeader(http.StatusAccepted)
					return
				}

				w.WriteHeader(http.StatusOK)
				fmt.Fprintln(w, `{"x": "y"}`)
			},
			"y",
			"",
		},
		{
			"accepted indefinitely",
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusAccepted)
			},
			"",
			"waited longer than MaxWait",
		},
	}

	MaxWait = 1 * time.Second

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			n = 0

			api := httptest.NewServer(http.HandlerFunc(tt.handler))
			defer api.Close()
			API = api.URL

			User = "x"
			Token = "x"
			var scan struct {
				X string `json:"x"`
			}
			_, err := Request(&scan, "GET", "/", nil)
			if !errorContains(err, tt.wantErr) {
				t.Fatal(err)
			}
			if scan.X != tt.want {
				t.Fatalf("scan.X wrong\nwant: %q\ngot:  %q", tt.want, scan.X)
			}
		})
	}
}

func TestPaginate(t *testing.T) {
	// TODO
}

func errorContains(out error, want string) bool {
	if out == nil {
		return want == ""
	}
	if want == "" {
		return false
	}
	return strings.Contains(out.Error(), want)
}
