package bump

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"path"
	"sync"
	"time"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/joe-at-startupmedia/version-bump/v2/console"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"golang.org/x/mod/semver"
)

func Run(action int) {
	// check for an update in parallel
	updateVersion := make(chan string, 1)
	updateVersionError := make(chan error, 1)
	var wg sync.WaitGroup
	wg.Add(1)
	go getLatestVersion(&wg, updateVersion, updateVersionError)

	dir := "."
	p, err := New(afero.NewOsFs(), osfs.New(path.Join(dir, ".git")), osfs.New(dir), dir, true)
	if err != nil {
		console.Fatal(errors.Wrap(err, "error preparing project configuration"))
	}

	if err := p.Bump(action); err != nil {
		console.Fatal(errors.Wrap(err, "error bumping a version"))
	}

	// notify user about an update
	wg.Wait()
	err = <-updateVersionError
	v := <-updateVersion
	if err != nil {
		console.ErrorCheckingForUpdate(err)
	} else if v != "" {
		console.UpdateAvailable(v)
	}
}

func getLatestVersion(wg *sync.WaitGroup, version chan string, resultErr chan error) {
	defer wg.Done()

	type response struct {
		TagName string `json:"tag_name"`
	}

	cli := &http.Client{
		Timeout: time.Second * 3,
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout: 2 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 5 * time.Second,
		},
	}

	res, err := cli.Get("https://api.github.com/repos/joe-at-startupmedia/version-bump/releases/latest")
	if err != nil {
		version <- ""
		resultErr <- err
		return
	}
	defer res.Body.Close()

	d := new(response)
	if err = json.NewDecoder(res.Body).Decode(d); err != nil {
		version <- ""
		resultErr <- err
		return
	}

	if semver.Compare(d.TagName, fmt.Sprintf("v%v", Version)) == 1 {
		version <- d.TagName
		resultErr <- nil
		return
	}

	version <- ""
	resultErr <- nil
}