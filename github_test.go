// +build testgithub

package hubhub

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func init() {
	User = os.Getenv("GITHUB_USER")
	Token = os.Getenv("GITHUB_PASS")
}

func TestRequest(t *testing.T) {
	t.Run("get repo", func(t *testing.T) {
		var r struct {
			ID int64 `json:"id"`
		}
		_, err := Request(&r, "GET", "/repos/Carpetsmoker/hubhub")
		if err != nil {
			t.Fatal(err)
		}

		if r.ID != 142667576 {
			t.Errorf("ID wrong: %v", r.ID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		var r struct {
			ID int64 `json:"id"`
		}
		_, err := Request(&r, "GET", "/repos/Carpetsmoker/will_never_exist")
		if err == nil {
			t.Fatal("err is not nil")
		}

		if r.ID != 0 {
			t.Errorf("ID wrong: %v", r.ID)
		}
	})
}

func TestRequestStat(t *testing.T) {
	var r []struct {
		Total int64 `json:"Total"`
	}
	err := RequestStat(&r, "GET", "/repos/Carpetsmoker/hubhub/stats/contributors", 300*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	if len(r) < 1 {
		t.Fatalf("0-legth in scan?")
	}

	if r[0].Total < 6 {
		t.Errorf("total wrong: %v", r[0].Total)
	}
}

func TestListRepos(t *testing.T) {
	repos, err := ListRepos("users/Carpetsmoker")
	if err != nil {
		t.Fatal(err)
	}

	if len(repos) < 50 {
		t.Fatalf("unexpected length: %v", len(repos))
	}
}

func TestPaginate(t *testing.T) {

	type repo struct {
		Name string `json:"name"`
	}

	t.Run("no-pointer", func(t *testing.T) {
		var repos []repo
		defer func() {
			r := recover()
			if r == nil {
				t.Errorf("no panic")
			}
		}()
		Paginate(repos, "", "", 0)
	})
	t.Run("noslice", func(t *testing.T) {
		var repos repo
		defer func() {
			r := recover()
			if r == nil {
				t.Errorf("no panic")
			}
		}()
		Paginate(repos, "", "", 0)
	})

	t.Run("2-pages", func(t *testing.T) {
		var repos []repo

		err := Paginate(&repos, "GET", "/users/Carpetsmoker/repos", 2)
		if err != nil {
			t.Fatal(err)
		}

		if len(repos) < 66 {
			t.Fatalf("unexpected length: %v", len(repos))
		}
	})

	t.Run("0-pages", func(t *testing.T) {
		var repos []repo

		err := Paginate(&repos, "GET", "/users/Carpetsmoker/repos", 0)
		if err != nil {
			t.Fatal(err)
		}

		fmt.Println(len(repos))
		fmt.Println(repos)

		if len(repos) < 66 {
			t.Fatalf("unexpected length: %v", len(repos))
		}
	})
}
