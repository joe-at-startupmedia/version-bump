package bump

import (
	"fmt"
	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/joe-at-startupmedia/version-bump/v2/gpg"
	"github.com/joe-at-startupmedia/version-bump/v2/version"
	"path"
	"regexp"
	"strings"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/joe-at-startupmedia/version-bump/v2/console"
	"github.com/joe-at-startupmedia/version-bump/v2/langs"
	"github.com/pelletier/go-toml/v2"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type versionBumpData struct {
	bump       *Bump
	versionMap map[string]int
	versionStr *string
	runArgs    *RunArgs
}

func New(fs afero.Fs, meta, data billy.Filesystem, dir string) (*Bump, error) {
	repo, err := git.Open(
		filesystem.NewStorage(meta, cache.NewObjectLRU(cache.DefaultMaxSize)),
		data,
	)
	if err != nil {
		return nil, errors.Wrap(err, "error opening repository")
	}
	localGitConfig, err := repo.ConfigScoped(config.GlobalScope)
	if err != nil {
		return nil, errors.Wrap(err, "error retrieving global git configuration")
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return nil, errors.Wrap(err, "error retrieving git worktree")
	}

	// NOTE: default config
	dirs := []string{dir}
	o := &Bump{
		FS: fs,
		Configuration: Configuration{
			Docker: Language{
				Enabled:     true,
				Directories: dirs,
			},
			Go: Language{
				Enabled:     true,
				Directories: dirs,
			},
			JavaScript: Language{
				Enabled:     true,
				Directories: dirs,
			},
		},
		Git: GitConfig{
			Repository: repo,
			Worktree:   worktree,
			Config:     localGitConfig,
		},
	}

	// check for config file
	content, err := readFile(fs, ".bump")
	if err != nil {
		// NOTE: return default settings if config file not found
		if strings.Contains(err.Error(), "no such file or directory") || strings.Contains(err.Error(), "file does not exist") {
			return o, nil
		} else {
			return nil, errors.Wrap(err, "error reading project config file")
		}
	}

	// parse config file
	userConfig := new(Configuration)
	if err := toml.Unmarshal([]byte(strings.Join(content, "\n")), userConfig); err != nil {
		return nil, errors.Wrap(err, "error parsing project config file")
	}

	o.Configuration = Configuration{
		Docker: Language{
			Enabled:     userConfig.Docker.Enabled,
			Directories: dirs,
		},
		Go: Language{
			Enabled:     userConfig.Go.Enabled,
			Directories: dirs,
		},
		JavaScript: Language{
			Enabled:     userConfig.JavaScript.Enabled,
			Directories: dirs,
		},
	}

	if len(userConfig.Docker.Directories) != 0 {
		o.Configuration.Docker.Directories = userConfig.Docker.Directories
	}

	if len(userConfig.Go.Directories) != 0 {
		o.Configuration.Go.Directories = userConfig.Go.Directories
	}

	if len(userConfig.JavaScript.Directories) != 0 {
		o.Configuration.JavaScript.Directories = userConfig.JavaScript.Directories
	}

	if len(userConfig.Docker.ExcludeFiles) != 0 {
		o.Configuration.Docker.ExcludeFiles = userConfig.Docker.ExcludeFiles
	}

	if len(userConfig.Go.ExcludeFiles) != 0 {
		o.Configuration.Go.ExcludeFiles = userConfig.Go.ExcludeFiles
	}

	if len(userConfig.JavaScript.ExcludeFiles) != 0 {
		o.Configuration.JavaScript.ExcludeFiles = userConfig.JavaScript.ExcludeFiles
	}

	return o, nil
}

func (b *Bump) Bump(ra *RunArgs) error {
	console.IncrementProjectVersion()

	versionMap := make(map[string]int)
	var newVersionStr string
	files := make([]string, 0)

	vbd := &versionBumpData{
		b,
		versionMap,
		&newVersionStr,
		ra,
	}

	if b.Configuration.Docker.Enabled {
		modifiedFiles, err := vbd.bumpComponent(langs.Docker, b.Configuration.Docker)
		if err != nil {
			return errors.Wrap(err, "error incrementing version in Docker project")
		}
		files = append(files, modifiedFiles...)
	}

	if b.Configuration.Go.Enabled {
		modifiedFiles, err := vbd.bumpComponent(langs.Go, b.Configuration.Go)
		if err != nil {
			return errors.Wrap(err, "error incrementing version in Go project")
		}
		files = append(files, modifiedFiles...)
	}

	if b.Configuration.JavaScript.Enabled {
		modifiedFiles, err := vbd.bumpComponent(langs.JavaScript, b.Configuration.JavaScript)
		if err != nil {
			return errors.Wrap(err, "error incrementing version in JavaScript project")
		}
		files = append(files, modifiedFiles...)
	}

	if len(versionMap) > 1 {
		return errors.New("inconsistent versioning")
	} else if len(versionMap) == 0 {
		return errors.New("0 files updated")
	}

	if len(files) != 0 {
		console.CommittingChanges()

		var gpgEntity *openpgp.Entity

		if ra.PassphrasePrompt != nil {
			gpgSigningKey, err := gpg.GetSigningKeyFromConfig(b.Git.Config)
			if err != nil {
				return errors.Wrap(err, "error retrieving gpg configuration")
			}
			if gpgSigningKey != "" {
				gpgEntity, err = vbd.passphrasePromptWithRetries(gpgSigningKey, 3, 0)
				if err != nil {
					return err
				}
			}
		}

		if err := b.Git.Save(files, newVersionStr, gpgEntity); err != nil {
			return errors.Wrap(err, "error committing changes")
		}
	}

	return nil
}

// +gocover:ignore:block
func (vbd *versionBumpData) passphrasePromptWithRetries(gpgSigningKey string, retryLimit int, retryCount int) (*openpgp.Entity, error) {
	if retryCount < retryLimit {
		keyPassphrase, err := vbd.runArgs.PassphrasePrompt()
		if err != nil {
			return nil, err
		}
		gpgEntity, err := gpg.GetGpgEntity(keyPassphrase, gpgSigningKey)
		if err != nil {
			return vbd.passphrasePromptWithRetries(gpgSigningKey, retryLimit, retryCount+1)
		} else {
			return gpgEntity, nil
		}
	} else {
		return nil, errors.New("could not validate gpg signing key")
	}
}

func (vbd *versionBumpData) bumpComponent(langName string, lang Language) ([]string, error) {
	console.Language(langName)
	files := make([]string, 0)

	for _, dir := range lang.Directories {
		f, err := getFiles(vbd.bump.FS, dir, lang.ExcludeFiles)
		if err != nil {
			return []string{}, errors.Wrap(err, "error listing directory files")
		}

		langSettings := langs.New(langName)
		if langSettings == nil {
			return []string{}, errors.New(fmt.Sprintf("not supported language: %v", langName))
		}

		modifiedFiles, err := vbd.incrementVersion(
			dir,
			filterFiles(langSettings.Files, f),
			*langSettings,
		)
		if err != nil {
			return []string{}, err
		}

		files = append(files, modifiedFiles...)
	}

	return files, nil
}

func (vbd *versionBumpData) incrementVersion(dir string, files []string, lang langs.Language) ([]string, error) {
	var identified bool
	modifiedFiles := make([]string, 0)

	for _, file := range files {
		filepath := path.Join(dir, file)
		fileContent, err := readFile(vbd.bump.FS, filepath)
		if err != nil {
			return []string{}, errors.Wrapf(err, "error reading a file %v", file)
		}
		var oldVersion *version.Version
		// get current version
		if lang.Regex != nil {
		outer:
			for _, line := range fileContent {
				for _, expression := range *lang.Regex {
					regex := regexp.MustCompile(expression)
					if regex.MatchString(line) {
						oldVersion, err = version.NewFromRegex(line, regex)
						if err != nil {
							return []string{}, errors.Wrapf(err, "error parsing semantic version at file %v from version: %s", filepath, oldVersion)
						}
						break outer
					}
				}
			}
		}

		if lang.JSONFields != nil {
			for _, field := range *lang.JSONFields {
				oldVersion, err = version.New(gjson.Get(strings.Join(fileContent, ""), field).String())
				if err != nil {
					return []string{}, errors.Wrapf(err, "error parsing semantic version at file %v", filepath)
				}
				break
			}
		}

		if oldVersion != nil {

			oldVersionStr := oldVersion.String()
			err = oldVersion.Increment(vbd.runArgs.VersionType, vbd.runArgs.PreReleaseType, vbd.runArgs.PreReleaseMetadata)
			if err != nil {
				return []string{}, errors.Wrapf(err, "error bumping version %v", filepath)
			}
			*vbd.versionStr = oldVersion.String()

			if strings.Compare(oldVersionStr, *vbd.versionStr) == 0 {
				//no changes in version
				continue
			}

			identified = true
			if vbd.runArgs.ConfirmationPrompt != nil {
				confirmed, err := vbd.runArgs.ConfirmationPrompt(oldVersionStr, *vbd.versionStr, file)
				if err != nil {
					return []string{}, errors.Wrap(err, "error during confirmation prompt")
				} else if !confirmed {
					//return []string{}, errors.New("proposed version was denied")
					//continue allows scenarios where denying changes in specific file(s) is necessary
					continue
				}
			}

			vbd.versionMap[oldVersionStr]++

			// set future version
			if lang.Regex != nil {
				newContent := make([]string, 0)

				for _, line := range fileContent {
					var added bool
					for _, expression := range *lang.Regex {
						regex := regexp.MustCompile(expression)
						if regex.MatchString(line) {
							l := strings.ReplaceAll(line, oldVersionStr, *vbd.versionStr)
							newContent = append(newContent, l)
							added = true
						}
					}

					if !added {
						newContent = append(newContent, line)
					}
				}

				newContent = append(newContent, "")
				if err = writeFile(vbd.bump.FS, filepath, strings.Join(newContent, "\n")); err != nil {
					return []string{}, errors.Wrapf(err, "error writing to file %v", filepath)
				}

				modifiedFiles = append(modifiedFiles, filepath)
			}

			if lang.JSONFields != nil {
				for _, field := range *lang.JSONFields {
					if gjson.Get(strings.Join(fileContent, ""), field).Exists() {
						newContent, err := sjson.Set(strings.Join(fileContent, "\n"), field, *vbd.versionStr)
						if err != nil {
							return []string{}, errors.Wrapf(err, "error setting new version on content of a file %v", file)
						}

						if err := writeFile(vbd.bump.FS, filepath, newContent); err != nil {
							return []string{}, errors.Wrapf(err, "error writing to file %v", filepath)
						}

						modifiedFiles = append(modifiedFiles, filepath)
					}
				}
			}

			if len(modifiedFiles) > 0 {
				fmt.Printf("Modified:%s\n", console.VersionUpdate(oldVersionStr, *vbd.versionStr, filepath))
			}
		}
	}

	if len(files) > 0 && !identified {
		console.Error("    Version was not identified")
	}

	return modifiedFiles, nil
}
