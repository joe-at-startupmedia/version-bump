package langs

import (
	"fmt"
	"github.com/joe-at-startupmedia/version-bump/v2/langs/docker"
	"github.com/joe-at-startupmedia/version-bump/v2/langs/golang"
	"github.com/joe-at-startupmedia/version-bump/v2/langs/js"
	"github.com/joe-at-startupmedia/version-bump/v2/version"
)

// DefaultSettings these settings can be overridden by Config
type DefaultSettings struct {
	Regex      *[]string
	JSONFields *[]string
	Name       string
	Files      []string
}

// Config value populated from the .bump file which override DefaultSettings
type Config struct {
	Regex        []string
	JSONFields   []string
	Name         string
	Files        []string
	Directories  []string
	ExcludeFiles []string `toml:"exclude_files"`
	Enabled      bool
}

// ConfigDecoder used to parse the .bump toml file
type ConfigDecoder struct {
	Generic    []Config
	Docker     Config
	Go         Config
	JavaScript Config
}

var Languages = []DefaultSettings{
	{
		Name:  docker.Name,
		Files: docker.Files,
		Regex: &docker.Regex,
	},
	{
		Name:  golang.Name,
		Files: golang.Files,
		Regex: &golang.Regex,
	},
	{
		Name:       js.Name,
		Files:      js.Files,
		JSONFields: &js.JSONFields,
	},
}

var Supported map[string]*DefaultSettings

func init() {
	Supported = make(map[string]*DefaultSettings, len(Languages))
	for li := range Languages {
		Supported[Languages[li].Name] = &Languages[li]
	}
}

func GetLanguageByName(langName string) *DefaultSettings {
	langSettings := Supported[langName]
	if langSettings == nil {
		langSettings = &DefaultSettings{
			Regex: &[]string{
				fmt.Sprintf("^.*?[\"']?(?P<version>%v)[\"']?", version.Regex),
			},
			Name: langName,
		}
	}
	return langSettings
}

func (c *Config) GetDirectories() []string {
	if len(c.Directories) == 0 {
		c.Directories = []string{"."}
	}
	return c.Directories
}
