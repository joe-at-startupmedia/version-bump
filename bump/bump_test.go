package bump_test

import (
	"fmt"
	"github.com/joe-at-startupmedia/version-bump/v2/langs"
	"github.com/joe-at-startupmedia/version-bump/v2/langs/docker"
	"github.com/joe-at-startupmedia/version-bump/v2/langs/golang"
	"github.com/joe-at-startupmedia/version-bump/v2/langs/js"
	"path"
	"reflect"
	"testing"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/joe-at-startupmedia/version-bump/v2/bump"
	"github.com/joe-at-startupmedia/version-bump/v2/git"
	"github.com/joe-at-startupmedia/version-bump/v2/mocks"
	"github.com/joe-at-startupmedia/version-bump/v2/version"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var testSuites map[string]testBumpTestSuite

func TestBump_New(t *testing.T) {
	a := assert.New(t)

	type configFile struct {
		Exists  bool
		Content string
	}

	type test struct {
		ConfigFile            configFile
		ExpectedConfiguration bump.Configuration
		ExpectedError         string
	}

	suites := map[string]test{
		"Automatic": {
			ConfigFile: configFile{},
			ExpectedConfiguration: bump.Configuration{
				langs.Config{
					Name:        docker.Name,
					Enabled:     true,
					Directories: []string{"."},
				},
				langs.Config{
					Name:        golang.Name,
					Enabled:     true,
					Directories: []string{"."},
				},
				langs.Config{
					Name:        js.Name,
					Enabled:     true,
					Directories: []string{"."},
				},
			},
			ExpectedError: "",
		},
		"Docker": {
			ConfigFile: configFile{
				Exists: true,
				Content: `[docker]
enabled = true
directories = ['dir1','dir2']`,
			},
			ExpectedConfiguration: bump.Configuration{
				langs.Config{
					Name:        docker.Name,
					Enabled:     true,
					Directories: []string{"dir1", "dir2"},
				},
			},
			ExpectedError: "",
		},
		"Go": {
			ConfigFile: configFile{
				Exists: true,
				Content: `[go]
enabled = true
directories = ['dir1','dir2']`,
			},
			ExpectedConfiguration: bump.Configuration{
				langs.Config{
					Name:        golang.Name,
					Enabled:     true,
					Directories: []string{"dir1", "dir2"},
				},
			},
			ExpectedError: "",
		},
		"JavaScript": {
			ConfigFile: configFile{
				Exists: true,
				Content: `[javascript]
enabled = true
directories = ['dir1','dir2']`,
			},
			ExpectedConfiguration: bump.Configuration{
				langs.Config{
					Name:        js.Name,
					Enabled:     true,
					Directories: []string{"dir1", "dir2"},
				},
			},
			ExpectedError: "",
		},
		"Complex": {
			ConfigFile: configFile{
				Exists: true,
				Content: `[docker]
enabled = true
directories = [ '.', 'tools/qa' ]
				
[go]
enabled = true
directories = [ 'server', 'tools/cli', 'tools/qa' ]
				
[javascript]
enabled = true
directories = [ 'client' ]`,
			},
			ExpectedConfiguration: bump.Configuration{
				langs.Config{
					Name:        docker.Name,
					Enabled:     true,
					Directories: []string{".", "tools/qa"},
				},
				langs.Config{
					Name:        golang.Name,
					Enabled:     true,
					Directories: []string{"server", "tools/cli", "tools/qa"},
				},
				langs.Config{
					Name:        js.Name,
					Enabled:     true,
					Directories: []string{"client"},
				},
			},
			ExpectedError: "",
		},
		"ComplexWithOneDisabled": {
			ConfigFile: configFile{
				Exists: true,
				Content: `[docker]
enabled = true
directories = [ '.', 'tools/qa' ]
				
[go]
enabled = true
directories = [ 'server', 'tools/cli', 'tools/qa' ]
				
[javascript]
enabled = false
directories = [ 'client' ]`,
			},
			ExpectedConfiguration: bump.Configuration{
				langs.Config{
					Name:        docker.Name,
					Enabled:     true,
					Directories: []string{".", "tools/qa"},
				},
				langs.Config{
					Name:        golang.Name,
					Enabled:     true,
					Directories: []string{"server", "tools/cli", "tools/qa"},
				},
			},
			ExpectedError: "",
		},
		"Exclude Files": {
			ConfigFile: configFile{
				Exists: true,
				Content: `[docker]
enabled = true
directories = [ '.', 'tools/qa' ]
exclude_files = [ 'tools/qa/Dockerfile' ]
				
[go]
enabled = true
directories = [ 'server', 'tools/cli', 'tools/qa' ]
exclude_files = [ 'tools/cli/main_test.go' ]
				
[javascript]
enabled = true
directories = [ 'client' ]
exclude_files = [ 'client/test.js' ]`,
			},
			ExpectedConfiguration: bump.Configuration{
				langs.Config{
					Name:         docker.Name,
					Enabled:      true,
					Directories:  []string{".", "tools/qa"},
					ExcludeFiles: []string{"tools/qa/Dockerfile"},
				},
				langs.Config{
					Name:         golang.Name,
					Enabled:      true,
					Directories:  []string{"server", "tools/cli", "tools/qa"},
					ExcludeFiles: []string{"tools/cli/main_test.go"},
				},
				langs.Config{
					Name:         js.Name,
					Enabled:      true,
					Directories:  []string{"client"},
					ExcludeFiles: []string{"client/test.js"},
				},
			},
			ExpectedError: "",
		},
	}

	var counter int
	for name, testSuite := range suites {
		counter++
		t.Logf("Test Case %v/%v - %s", counter, len(suites), name)
		fs := afero.NewMemMapFs()
		meta := memfs.New()
		data := memfs.New()

		err := git.Init(meta, data)

		if err != nil {
			t.Errorf("error preparing test case: error initializing repository: %v", err)
			continue
		}

		if testSuite.ConfigFile.Exists {
			f, err := fs.Create(".bump")
			if err != nil {
				t.Errorf("error preparing test case: error creating Docker files: %v", err)
				continue
			}

			_, err = f.WriteString(testSuite.ConfigFile.Content)
			if err != nil {
				t.Errorf("error preparing test case: error writing Docker files: %v", err)
				continue
			}
		}

		b, err := bump.From(fs, meta, data, ".")
		if testSuite.ExpectedError != "" || err != nil {
			a.EqualError(err, testSuite.ExpectedError)
			a.Equal(nil, b)
		} else {
			a.Equal(testSuite.ExpectedConfiguration, b.Configuration)
			a.NotEqual(nil, b.Git)
		}
	}
}

type file struct {
	Name                string
	ExpectedToBeChanged bool
	Content             string
}

type fileMap map[string][]file

type allFiles struct {
	Docker     fileMap
	Go         fileMap
	JavaScript fileMap
}

type testBumpTestSuite struct {
	Version            string
	Configuration      bump.Configuration
	Files              allFiles
	VersionType        version.Type
	PreReleaseType     version.PreReleaseType
	MockAddError       error
	MockCommitError    error
	MockCreateTagError error
	ExpectedError      string
}

func TestBump_Bump(t *testing.T) {
	a := assert.New(t)

	testSuites = map[string]testBumpTestSuite{
		"Empty Configuration": {
			Version:            "",
			Configuration:      bump.Configuration{},
			Files:              allFiles{},
			VersionType:        version.Major,
			PreReleaseType:     version.NotAPreRelease,
			MockAddError:       nil,
			MockCommitError:    nil,
			MockCreateTagError: nil,
			ExpectedError:      "0 files updated",
		},
		"Docker - Single, without Quotes": {
			Version: "2.0.0",
			Configuration: bump.Configuration{
				langs.Config{
					Name:        docker.Name,
					Enabled:     true,
					Directories: []string{"."},
				},
			},
			Files: allFiles{
				Docker: map[string][]file{
					".": {
						{
							Name:                "Dockerfile",
							ExpectedToBeChanged: true,
							Content: `FROM golang:1.16 as builder
WORKDIR /opt/src
COPY . .
RUN groupadd -g 1000 appuser &&\
	useradd -m -u 1000 -g appuser appuser

RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /opt/app
FROM scratch
LABEL "repository"="https://github.com/anton-yurchenko/git-release"
LABEL "maintainer"="Anton Yurchenko <anton.doar@gmail.com>"
LABEL org.opencontainers.image.version=1.2.3
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY LICENSE.md /LICENSE.md
COPY --from=builder --chown=1000:0 /opt/app /app
ENTRYPOINT [ "/app" ]`,
						},
					},
				},
			},
			VersionType:        version.Major,
			PreReleaseType:     version.NotAPreRelease,
			MockAddError:       nil,
			MockCommitError:    nil,
			MockCreateTagError: nil,
			ExpectedError:      "",
		},
		"Docker - Single, with Quotes": {
			Version: "2.0.0",
			Configuration: bump.Configuration{
				langs.Config{
					Name:        docker.Name,
					Enabled:     true,
					Directories: []string{"."},
				},
			},
			Files: allFiles{
				Docker: map[string][]file{
					".": {
						{
							Name:                "Dockerfile",
							ExpectedToBeChanged: true,
							Content: `FROM golang:1.16 as builder
WORKDIR /opt/src
COPY . .
RUN groupadd -g 1000 appuser &&\
	useradd -m -u 1000 -g appuser appuser

RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /opt/app
FROM scratch
LABEL "repository"="https://github.com/anton-yurchenko/git-release"
LABEL "maintainer"="Anton Yurchenko <anton.doar@gmail.com>"
LABEL "org.opencontainers.image.version"="v1.2.3"
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY LICENSE.md /LICENSE.md
COPY --from=builder --chown=1000:0 /opt/app /app
ENTRYPOINT [ "/app" ]`,
						},
					},
				},
			},
			VersionType:        version.Major,
			PreReleaseType:     version.NotAPreRelease,
			MockAddError:       nil,
			MockCommitError:    nil,
			MockCreateTagError: nil,
			ExpectedError:      "",
		},
		"Docker - Multiple, with Quotes": {
			Version: "4.0.0",
			Configuration: bump.Configuration{
				langs.Config{
					Name:        docker.Name,
					Enabled:     true,
					Directories: []string{"."},
				},
			},
			Files: allFiles{
				Docker: map[string][]file{
					".": {
						{
							Name:                "Dockerfile",
							ExpectedToBeChanged: true,
							Content: `FROM golang:1.16 as builder
WORKDIR /opt/src
COPY . .
RUN groupadd -g 1000 appuser &&\
	useradd -m -u 1000 -g appuser appuser

RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /opt/app
FROM scratch
LABEL "maintainer"="Anton Yurchenko <anton.doar@gmail.com>"
LABEL "repository"="https://github.com/anton-yurchenko/git-release" "org.opencontainers.image.version"="V3.4.7"
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY LICENSE.md /LICENSE.md
COPY --from=builder --chown=1000:0 /opt/app /app
ENTRYPOINT [ "/app" ]`,
						},
					},
				},
			},
			VersionType:        version.Major,
			PreReleaseType:     version.NotAPreRelease,
			MockAddError:       nil,
			MockCommitError:    nil,
			MockCreateTagError: nil,
			ExpectedError:      "",
		},
		"Docker - Multiple, without Quotes,": {
			Version: "2.0.0",
			Configuration: bump.Configuration{
				langs.Config{
					Name:        docker.Name,
					Enabled:     true,
					Directories: []string{"."},
				},
			},
			Files: allFiles{
				Docker: map[string][]file{
					".": {
						{
							Name:                "Dockerfile",
							ExpectedToBeChanged: true,
							Content: `FROM golang:1.16 as builder
WORKDIR /opt/src
COPY . .
RUN groupadd -g 1000 appuser &&\
	useradd -m -u 1000 -g appuser appuser

RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /opt/app
FROM scratch
LABEL "maintainer"="Anton Yurchenko <anton.doar@gmail.com>"
LABEL "repository"="https://github.com/anton-yurchenko/git-release" org.opencontainers.image.version="1.2.3"
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY LICENSE.md /LICENSE.md
COPY --from=builder --chown=1000:0 /opt/app /app
ENTRYPOINT [ "/app" ]`,
						},
					},
				},
			},
			VersionType:        version.Major,
			PreReleaseType:     version.NotAPreRelease,
			MockAddError:       nil,
			MockCommitError:    nil,
			MockCreateTagError: nil,
			ExpectedError:      "",
		},
		"Docker - Multi-line, with Quotes": {
			Version: "2.0.0",
			Configuration: bump.Configuration{
				langs.Config{
					Name:        docker.Name,
					Enabled:     true,
					Directories: []string{"."},
				},
			},
			Files: allFiles{
				Docker: map[string][]file{
					".": {
						{
							Name:                "Dockerfile",
							ExpectedToBeChanged: true,
							Content: `FROM golang:1.16 as builder
WORKDIR /opt/src
COPY . .
RUN groupadd -g 1000 appuser &&\
	useradd -m -u 1000 -g appuser appuser

RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /opt/app
FROM scratch
LABEL "repository"="https://github.com/anton-yurchenko/git-release" \
	"org.opencontainers.image.version"="v1.2.3" \
	"maintainer"="Anton Yurchenko <anton.doar@gmail.com>"
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY LICENSE.md /LICENSE.md
COPY --from=builder --chown=1000:0 /opt/app /app
ENTRYPOINT [ "/app" ]`,
						},
					},
				},
			},
			VersionType:        version.Major,
			PreReleaseType:     version.NotAPreRelease,
			MockAddError:       nil,
			MockCommitError:    nil,
			MockCreateTagError: nil,
			ExpectedError:      "",
		},
		"Docker - Multi-line, without Quotes": {
			Version: "2.0.0",
			Configuration: bump.Configuration{
				langs.Config{
					Name:        docker.Name,
					Enabled:     true,
					Directories: []string{"."},
				},
			},
			Files: allFiles{
				Docker: map[string][]file{
					".": {
						{
							Name:                "Dockerfile",
							ExpectedToBeChanged: true,
							Content: `FROM golang:1.16 as builder
WORKDIR /opt/src
COPY . .
RUN groupadd -g 1000 appuser &&\
	useradd -m -u 1000 -g appuser appuser

RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /opt/app
FROM scratch
LABEL "repository"="https://github.com/anton-yurchenko/git-release" \
org.opencontainers.image.version="v1.2.3" \
	"maintainer"="Anton Yurchenko <anton.doar@gmail.com>"
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY LICENSE.md /LICENSE.md
COPY --from=builder --chown=1000:0 /opt/app /app
ENTRYPOINT [ "/app" ]`,
						},
					},
				},
			},
			VersionType:        version.Major,
			PreReleaseType:     version.NotAPreRelease,
			MockAddError:       nil,
			MockCommitError:    nil,
			MockCreateTagError: nil,
			ExpectedError:      "",
		},
		"Go - Single Constant": {
			Version: "1.3.0",
			Configuration: bump.Configuration{
				langs.Config{
					Name:        golang.Name,
					Enabled:     true,
					Directories: []string{"."},
				},
			},
			Files: allFiles{
				Go: map[string][]file{
					".": {
						{
							Name:                "main.go",
							ExpectedToBeChanged: true,
							Content: `package main

import "fmt"

const Version string = "1.2.3"

func main() {
	fmt.Println(Version)
}`,
						},
					},
				},
			},
			VersionType:        version.Minor,
			PreReleaseType:     version.NotAPreRelease,
			MockAddError:       nil,
			MockCommitError:    nil,
			MockCreateTagError: nil,
			ExpectedError:      "",
		},
		"Go - Single Constant #2": {
			Version: "1.2.4",
			Configuration: bump.Configuration{
				langs.Config{
					Name:        golang.Name,
					Enabled:     true,
					Directories: []string{"."},
				},
			},
			Files: allFiles{
				Go: map[string][]file{
					".": {
						{
							Name:                "main.go",
							ExpectedToBeChanged: true,
							Content: `package main

import "fmt"

const Version := "1.2.3"

func main() {
	fmt.Println(Version)
}`,
						},
					},
				},
			},
			VersionType:        version.Patch,
			PreReleaseType:     version.NotAPreRelease,
			MockAddError:       nil,
			MockCommitError:    nil,
			MockCreateTagError: nil,
			ExpectedError:      "",
		},
		"Go - Multiple Constants": {
			Version: "2.0.0",
			Configuration: bump.Configuration{
				langs.Config{
					Name:        golang.Name,
					Enabled:     true,
					Directories: []string{"."},
				},
			},
			Files: allFiles{
				Go: map[string][]file{
					".": {
						{
							Name:                "main.go",
							ExpectedToBeChanged: true,
							Content: `package main

import "fmt"

const (
	Version                                          string = "1.2.4"
	SomeVeryLongVariableNameThatAddsALotOfWhitespace string = "abc"
)

func main() {
	fmt.Println(Version)
}`,
						},
					},
				},
			},
			VersionType:        version.Major,
			PreReleaseType:     version.NotAPreRelease,
			MockAddError:       nil,
			MockCommitError:    nil,
			MockCreateTagError: nil,
			ExpectedError:      "",
		},
		"JavaScript - Multiple Constants": {
			Version: "2.0.0",
			Configuration: bump.Configuration{
				langs.Config{
					Name:        js.Name,
					Enabled:     true,
					Directories: []string{"."},
				},
			},
			Files: allFiles{
				JavaScript: map[string][]file{
					".": {
						{
							Name:                "package.json",
							ExpectedToBeChanged: true,
							Content: `{
	"name": "git-release",
	"version": "1.2.3",
	"description": "A GitHub VersionType for creating a GitHub Release with Assets and Changelog whenever a new Tag is pushed to the repository.",
	"main": "wrapper.js",
	"directories": {
	  "doc": "docs"
	},
	"repository": {
	  "type": "git",
	  "url": "git+https://github.com/anton-yurchenko/git-release.git"
	},
	"keywords": [],
	"author": "Anton Yurchenko",
	"license": "MIT",
	"bugs": {
	  "url": "https://github.com/anton-yurchenko/git-release/issues"
	},
	"homepage": "https://github.com/anton-yurchenko/git-release#readme",
	"dependencies": {
	  "@actions/core": "^1.4.0"
	},
	"devDependencies": {}
}`,
						},
						{
							Name:                "package-lock.json",
							ExpectedToBeChanged: true,
							Content: `{
	"name": "git-release",
	"version": "1.2.3",
	"lockfileVersion": 2,
	"requires": true,
	"packages": {
	  "": {
		"version": "1.2.3",
		"license": "MIT",
		"dependencies": {
		  "@actions/core": "^1.4.0"
		},
		"devDependencies": {}
	  },
	  "node_modules/@actions/core": {
		"version": "1.4.0",
		"resolved": "https://registry.npmjs.org/@actions/core/-/core-1.4.0.tgz",
		"integrity": "sha512-CGx2ilGq5i7zSLgiiGUtBCxhRRxibJYU6Fim0Q1Wg2aQL2LTnF27zbqZOrxfvFQ55eSBW0L8uVStgtKMpa0Qlg=="
	  }
	},
	"dependencies": {
	  "@actions/core": {
		"version": "1.4.0",
		"resolved": "https://registry.npmjs.org/@actions/core/-/core-1.4.0.tgz",
		"integrity": "sha512-CGx2ilGq5i7zSLgiiGUtBCxhRRxibJYU6Fim0Q1Wg2aQL2LTnF27zbqZOrxfvFQ55eSBW0L8uVStgtKMpa0Qlg=="
	  }
	}
}`,
						},
					},
				},
			},
			VersionType:        version.Major,
			PreReleaseType:     version.NotAPreRelease,
			MockAddError:       nil,
			MockCommitError:    nil,
			MockCreateTagError: nil,
			ExpectedError:      "",
		},
		"Docker - Get Files Error": {
			Version: "2.0.0",
			Configuration: bump.Configuration{
				langs.Config{
					Name:        docker.Name,
					Enabled:     true,
					Directories: []string{"dir"},
				},
			},
			Files:              allFiles{},
			VersionType:        version.Major,
			PreReleaseType:     version.NotAPreRelease,
			MockAddError:       nil,
			MockCommitError:    nil,
			MockCreateTagError: nil,
			ExpectedError:      "error incrementing version in Docker project: error listing directory files: open dir: file does not exist",
		},
		"Go - Get Files Error": {
			Version: "2.0.0",
			Configuration: bump.Configuration{
				langs.Config{
					Name:        golang.Name,
					Enabled:     true,
					Directories: []string{"dir"},
				},
			},
			Files:              allFiles{},
			VersionType:        version.Major,
			PreReleaseType:     version.NotAPreRelease,
			MockAddError:       nil,
			MockCommitError:    nil,
			MockCreateTagError: nil,
			ExpectedError:      "error incrementing version in Go project: error listing directory files: open dir: file does not exist",
		},
		"JavaScript - Get Files Error": {
			Version: "2.0.0",
			Configuration: bump.Configuration{
				langs.Config{
					Name:        js.Name,
					Enabled:     true,
					Directories: []string{"dir"},
				},
			},
			Files:              allFiles{},
			VersionType:        version.Major,
			PreReleaseType:     version.NotAPreRelease,
			MockAddError:       nil,
			MockCommitError:    nil,
			MockCreateTagError: nil,
			ExpectedError:      "error incrementing version in JavaScript project: error listing directory files: open dir: file does not exist",
		},
		"Inconsistent Versioning": {
			Version: "2.0.0",
			Configuration: bump.Configuration{
				langs.Config{
					Name:        docker.Name,
					Enabled:     true,
					Directories: []string{"."},
				},
				langs.Config{
					Name:        golang.Name,
					Enabled:     true,
					Directories: []string{"."},
				},
			},
			Files: allFiles{
				Docker: map[string][]file{
					".": {
						{
							Name:                "Dockerfile",
							ExpectedToBeChanged: true,
							Content: `FROM golang:1.16 as builder
WORKDIR /opt/src
COPY . .
RUN groupadd -g 1000 appuser &&\
	useradd -m -u 1000 -g appuser appuser

RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /opt/app
FROM scratch
LABEL "repository"="https://github.com/anton-yurchenko/git-release"
LABEL "maintainer"="Anton Yurchenko <anton.doar@gmail.com>"
LABEL "org.opencontainers.image.version"="1.2.3"
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY LICENSE.md /LICENSE.md
COPY --from=builder --chown=1000:0 /opt/app /app
ENTRYPOINT [ "/app" ]`,
						},
					},
				},
				Go: map[string][]file{
					".": {
						{
							Name:                "main.go",
							ExpectedToBeChanged: true,
							Content: `package main

import "fmt"

const Version string = "1.3.0"

func main() {
	fmt.Println(Version)
}`,
						},
					},
				},
			},
			VersionType:        version.Major,
			PreReleaseType:     version.NotAPreRelease,
			MockAddError:       nil,
			MockCommitError:    nil,
			MockCreateTagError: nil,
			ExpectedError:      "inconsistent versioning",
		},
		"Save Error": {
			Version: "2.0.0",
			Configuration: bump.Configuration{
				langs.Config{
					Name:        docker.Name,
					Enabled:     true,
					Directories: []string{"."},
				},
			},
			Files: allFiles{
				Docker: map[string][]file{
					".": {
						{
							Name:                "Dockerfile",
							ExpectedToBeChanged: true,
							Content: `FROM golang:1.16 as builder
WORKDIR /opt/src
COPY . .
RUN groupadd -g 1000 appuser &&\
	useradd -m -u 1000 -g appuser appuser

RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /opt/app
FROM scratch
LABEL "repository"="https://github.com/anton-yurchenko/git-release"
LABEL "maintainer"="Anton Yurchenko <anton.doar@gmail.com>"
LABEL org.opencontainers.image.version 1.2.3
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY LICENSE.md /LICENSE.md
COPY --from=builder --chown=1000:0 /opt/app /app
ENTRYPOINT [ "/app" ]`,
						},
					},
				},
			},
			VersionType:        version.Major,
			PreReleaseType:     version.NotAPreRelease,
			MockAddError:       nil,
			MockCommitError:    errors.New("reason"),
			MockCreateTagError: nil,
			ExpectedError:      "error committing changes: error committing changes: reason",
		},
		"Exclude Files": {
			Version: "2.0.0",
			Configuration: bump.Configuration{
				langs.Config{
					Name:        docker.Name,
					Enabled:     true,
					Directories: []string{"."},
				},
				langs.Config{
					Name:         golang.Name,
					Enabled:      true,
					Directories:  []string{".", "lib"},
					ExcludeFiles: []string{"lib/lib_test.go"},
				},
			},
			Files: allFiles{
				Docker: map[string][]file{
					".": {
						{
							Name:                "Dockerfile",
							ExpectedToBeChanged: true,
							Content: `FROM golang:1.16 as builder
WORKDIR /opt/src
COPY . .
RUN groupadd -g 1000 appuser &&\
	useradd -m -u 1000 -g appuser appuser

RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /opt/app
FROM scratch
LABEL "repository"="https://github.com/anton-yurchenko/git-release"
LABEL "maintainer"="Anton Yurchenko <anton.doar@gmail.com>"
LABEL "version"="1.2.3"
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY LICENSE.md /LICENSE.md
COPY --from=builder --chown=1000:0 /opt/app /app
ENTRYPOINT [ "/app" ]`,
						},
					},
				},
				Go: map[string][]file{
					".": {
						{
							Name:                "main.go",
							ExpectedToBeChanged: true,
							Content: `package main

import "fmt"
							
const Version string = "1.2.3"
							
func main() {
	fmt.Println(Version)
}`,
						},
					},
					"lib": {
						{
							Name:                "lib.go",
							ExpectedToBeChanged: true,
							Content: `package lib

import "fmt"

const Version string = "1.2.3"`,
						},
						{
							Name:                "lib_test.go",
							ExpectedToBeChanged: false,
							Content: `package lib_test

import "fmt"

const Version string = "1.2.3"`,
						},
					},
				},
			},
			VersionType:        version.Major,
			PreReleaseType:     version.NotAPreRelease,
			MockAddError:       nil,
			MockCommitError:    nil,
			MockCreateTagError: nil,
			ExpectedError:      "",
		},
	}

	var counter int
	for name, testSuite := range testSuites {
		counter++
		t.Logf("Test Case %v/%v - %s", counter, len(testSuites), name)

		_, err := runBumpTest(t, testSuite, &bump.RunArgs{
			VersionType:    testSuite.VersionType,
			PreReleaseType: testSuite.PreReleaseType,
		})

		if testSuite.ExpectedError != "" || err != nil {
			a.EqualError(err, testSuite.ExpectedError)
		}
	}
}

func TestBump_WithVanillaFsRepoDoesntExist(t *testing.T) {
	a := assert.New(t)
	_, err := bump.New(".")
	a.ErrorContains(err, "error opening repository: repository does not exist")
}

func TestBump_BrokenBumpFile(t *testing.T) {
	a := assert.New(t)
	fs := afero.NewMemMapFs()
	meta := memfs.New()
	data := memfs.New()
	_ = git.Init(meta, data)
	f, err := fs.Create(".bump")
	_, err = f.WriteString("brokenbump-contents")
	_, err = bump.From(fs, meta, data, ".")
	a.ErrorContains(err, "error parsing project config file")
}

func TestBump_ConfirmationError(t *testing.T) {
	a := assert.New(t)

	testSuite := testSuites["Go - Single Constant #2"]

	_, err := runBumpTest(t, testSuite, &bump.RunArgs{
		VersionType:    testSuite.VersionType,
		PreReleaseType: testSuite.PreReleaseType,
		ConfirmationPrompt: func(_ string, _ string, _ string) (bool, error) {
			return true, fmt.Errorf("confirmation_error")
		},
	})
	a.ErrorContains(err, "error during confirmation prompt: confirmation_error")
}

func TestBump_ConfirmationDenied(t *testing.T) {
	a := assert.New(t)

	testSuite := testSuites["Go - Single Constant #2"]

	_, err := runBumpTest(t, testSuite, &bump.RunArgs{
		VersionType:    testSuite.VersionType,
		PreReleaseType: testSuite.PreReleaseType,
		ConfirmationPrompt: func(_ string, _ string, _ string) (bool, error) {
			return false, nil
		},
	})
	//currently we continue through the loop instead of returning an error
	a.ErrorContains(err, "0 files updated")
}

func GitConfigParserMockScenarioOne() *mocks.GitConfigParser {
	cpm := new(mocks.GitConfigParser)
	cpm.On("SetConfig", mock.AnythingOfType("*config.Config")).Return(nil)
	cpm.On("GetSectionOption", "commit", "gpgsign").Return("true")
	cpm.On("GetSectionOption", "user", "signingkey").Return("ACB2CCCDA93C90BF")
	return cpm
}

func TestBump_PassphraseError(t *testing.T) {
	a := assert.New(t)

	testSuite := testSuites["Go - Single Constant #2"]
	gcp := GitConfigParserMockScenarioOne()
	defer gcp.AssertExpectations(t)
	bump.GitConfigParser = gcp

	_, err := runBumpTest(t, testSuite, &bump.RunArgs{
		VersionType:    testSuite.VersionType,
		PreReleaseType: testSuite.PreReleaseType,
		PassphrasePrompt: func() (string, error) {
			return "", fmt.Errorf("custom_passphrase_err")
		},
	})
	//currently we continue through the loop instead of returning an error
	a.ErrorContains(err, "custom_passphrase_err")
}

func TestBump_PassphraseGetSigningKeyError(t *testing.T) {
	a := assert.New(t)

	testSuite := testSuites["Go - Single Constant #2"]
	gcp := GitConfigParserMockScenarioOne()
	defer gcp.AssertExpectations(t)
	bump.GitConfigParser = gcp

	_, err := runBumpTest(t, testSuite, &bump.RunArgs{
		VersionType:    testSuite.VersionType,
		PreReleaseType: testSuite.PreReleaseType,
		PassphrasePrompt: func() (string, error) {
			return "", nil
		},
	})
	a.ErrorContains(err, "could not validate gpg signing key")
}

func TestBump_PassphraseGetGpgEntityError(t *testing.T) {
	a := assert.New(t)

	testSuite := testSuites["Go - Single Constant #2"]

	gcp := GitConfigParserMockScenarioOne()
	defer gcp.AssertExpectations(t)
	bump.GitConfigParser = gcp
	gea := new(mocks.GpgEntityAccessor)
	defer gea.AssertExpectations(t)
	gea.On("GetEntity", "", mock.Anything).Return(nil, errors.New("gpg_entity_error"))
	bump.GpgEntityAccessor = gea

	_, err := runBumpTest(t, testSuite, &bump.RunArgs{
		VersionType:    testSuite.VersionType,
		PreReleaseType: testSuite.PreReleaseType,
		PassphrasePrompt: func() (string, error) {
			return "", nil
		},
	})
	//currently we continue through the loop instead of returning an error
	a.ErrorContains(err, "could not validate gpg signing key")
}

func TestBump_PassphraseGetGpgDoesntError(t *testing.T) {
	a := assert.New(t)

	testSuite := testSuites["Go - Single Constant #2"]

	gcp := GitConfigParserMockScenarioOne()
	defer gcp.AssertExpectations(t)
	bump.GitConfigParser = gcp
	gea := new(mocks.GpgEntityAccessor)
	defer gea.AssertExpectations(t)
	gea.On("GetEntity", "", mock.Anything).Return(nil, nil)
	bump.GpgEntityAccessor = gea

	_, err := runBumpTest(t, testSuite, &bump.RunArgs{
		VersionType:    testSuite.VersionType,
		PreReleaseType: testSuite.PreReleaseType,
		PassphrasePrompt: func() (string, error) {
			return "", nil
		},
	})
	//currently we continue through the loop instead of returning an error
	a.Empty(err)
}

func TestBump_PassphraseGetShouldNotSign(t *testing.T) {
	a := assert.New(t)

	testSuite := testSuites["Go - Single Constant #2"]

	gcp := new(mocks.GitConfigParser)
	defer gcp.AssertExpectations(t)
	gcp.On("SetConfig", mock.AnythingOfType("*config.Config")).Return(nil)
	gcp.On("GetSectionOption", "commit", "gpgsign").Return("false")
	bump.GitConfigParser = gcp

	_, err := runBumpTest(t, testSuite, &bump.RunArgs{
		VersionType:    testSuite.VersionType,
		PreReleaseType: testSuite.PreReleaseType,
		PassphrasePrompt: func() (string, error) {
			return "", nil
		},
	})
	//currently we continue through the loop instead of returning an error
	a.Empty(err)
}

func TestBump_PassphraseGetShouldNotSignLoadConfigFails(t *testing.T) {
	a := assert.New(t)

	testSuite := testSuites["Go - Single Constant #2"]

	gcp := new(mocks.GitConfigParser)
	defer gcp.AssertExpectations(t)
	gcp.On("SetConfig", mock.AnythingOfType("*config.Config")).Return(nil)
	gcp.On("GetSectionOption", "commit", "gpgsign").Return("true")
	gcp.On("GetSectionOption", "user", "signingkey").Return("")
	gcp.On("LoadConfig", config.GlobalScope).Return(nil, errors.New("mock_test_load_config_error"))
	bump.GitConfigParser = gcp

	_, err := runBumpTest(t, testSuite, &bump.RunArgs{
		VersionType:    testSuite.VersionType,
		PreReleaseType: testSuite.PreReleaseType,
		PassphrasePrompt: func() (string, error) {
			return "", nil
		},
	})
	//currently we continue through the loop instead of returning an error
	a.ErrorContains(err, "error loading git configuration from global scope: mock_test_load_config_error")
}

func TestBump_PassphraseGetShouldNotSignLoadConfigPasses(t *testing.T) {
	a := assert.New(t)

	testSuite := testSuites["Go - Single Constant #2"]

	gcp := new(mocks.GitConfigParser)
	defer gcp.AssertExpectations(t)
	gcp.On("SetConfig", mock.AnythingOfType("*config.Config")).Return(nil)
	gcp.On("GetSectionOption", "commit", "gpgsign").Return("true")
	gcp.On("GetSectionOption", "user", "signingkey").Return("")
	gcp.On("LoadConfig", config.GlobalScope).Return(config.NewConfig(), nil)
	bump.GitConfigParser = gcp

	_, err := runBumpTest(t, testSuite, &bump.RunArgs{
		VersionType:    testSuite.VersionType,
		PreReleaseType: testSuite.PreReleaseType,
		PassphrasePrompt: func() (string, error) {
			return "", nil
		},
	})
	//currently we continue through the loop instead of returning an error
	a.Nil(err)
}

func runBumpTest(t *testing.T, testSuite testBumpTestSuite, ra *bump.RunArgs) (*bump.Bump, error) {

	m1 := new(mocks.Repository)
	m2 := new(mocks.Worktree)

	gitConfig := &config.Config{}
	gitConfig.User.Name = git.Username
	gitConfig.User.Email = git.Email

	r := bump.Bump{
		FS: afero.NewMemMapFs(),
		Git: &git.Instance{
			Config:     gitConfig,
			Repository: m1,
			Worktree:   m2,
		},
		Configuration: testSuite.Configuration,
	}

	shouldBeCommitted := false

	tfr := reflect.ValueOf(testSuite.Files)

	for i := range testSuite.Configuration {
		langConfig := testSuite.Configuration[i]

		sf := tfr.FieldByName(langConfig.Name)
		langFileMap := sf.Interface().(fileMap)

		if langConfig.Enabled {
			for _, dir := range langConfig.Directories {
				for tgtDir, tgtFiles := range langFileMap {
					if dir == tgtDir {
						for _, tgtFile := range tgtFiles {
							shouldBeCommitted = true
							f, err := r.FS.Create(path.Join(dir, tgtFile.Name))
							if err != nil {
								t.Errorf("error preparing test case: error creating %s files: %v", langConfig.Name, err)
								continue
							}

							_, err = f.WriteString(tgtFile.Content)
							if err != nil {
								t.Errorf("error preparing test case: error writing %s files: %v", langConfig.Name, err)
								continue
							}

							if tgtFile.ExpectedToBeChanged {
								var f string
								if dir == "." {
									f = tgtFile.Name
								} else {
									f = path.Join(dir, tgtFile.Name)
								}
								m2.On("Add", f).Return(nil, testSuite.MockAddError).Once()
							}
						}
					}
				}
			}
		}
	}

	if shouldBeCommitted {

		hash := plumbing.NewHash("abc")

		m2.On(
			"Commit", testSuite.Version, mock.AnythingOfType("*git.CommitOptions"),
		).Return(hash, testSuite.MockCommitError).Once()

		m1.On(
			"CreateTag", fmt.Sprintf("v%v", testSuite.Version), hash, mock.AnythingOfType("*git.CreateTagOptions"),
		).Return(nil, testSuite.MockCreateTagError).Once()
	}

	err := r.Bump(ra)

	return &r, err
}
