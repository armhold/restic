package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
)

// Config contains all the persistent configuration for the repos managed by the web interface
type Config struct {
	Repos    []*Repo `json:"Repos"`
	Filepath string `json:"-"`
}

type Repo struct {
	Name        string      `json:"Name"`     // "local repo"
	Path        string      `json:"Path"`     //  "b2:bucket-Name/Path"
	Password    string      `json:"Password"` // TODO: encrypt?
	BackupPaths *BackupPaths `json:"BackupPaths"`
}

type BackupPaths struct {
	Paths    map[string]bool `json:"Paths"`
	Excludes map[string]bool `json:"Excludes"`
}

func (b *BackupPaths) GetPaths() (result []string) {
	for k, _ := range b.Paths {
		result = append(result, k)
	}

	return
}

func (b *BackupPaths) GetExcludes() (result []string) {
	for k, _ := range b.Excludes {
		result = append(result, k)
	}

	return
}

func NewRepo(name, path, password string) *Repo {
	return &Repo{Name: name,
		Path: path,
		Password: password,
		BackupPaths: NewBackupPaths(),
	}
}


func NewBackupPaths() *BackupPaths {
	return &BackupPaths{
		Paths:    make(map[string]bool),
		Excludes: make(map[string]bool),
	}
}

func (c *Config) listRepos() []*Repo {
	var result []*Repo

	return result
}

func (c *Config) AddRepo(repo *Repo) error {
	for _, r := range c.Repos {
		if r.Name == repo.Name {
			return fmt.Errorf("repo with name \"%s\" already exists", repo.Name)
		}

		if r.Path == repo.Path {
			return fmt.Errorf("repo with path \"%s\" already exists", repo.Path)
		}
	}

	c.Repos = append(c.Repos, repo)
	return c.Save()
}

func (c *Config) Save() error {
	if c.Filepath == "" {
		return errors.New("filepath missing")
	}

	b, err := json.Marshal(c)
	if err != nil {
		return err
	}

	fmt.Printf("saving config to %s\n", c.Filepath)
	return ioutil.WriteFile(c.Filepath, b, 0600)
}

func LoadConfigFromDefault() (Config, error) {
	path, err := defaultConfigPath()
	if err != nil {
		return Config{}, err
	}

	result, err := LoadConfig(path)
	if os.IsNotExist(err) {
		err = result.Save()
	}

	return result, err
}

func LoadConfig(filepath string) (Config, error) {
	result := Config{Filepath: filepath}

	b, err := ioutil.ReadFile(filepath)
	if err != nil {
		return result, err
	}

	err = json.Unmarshal(b, &result)
	return result, err
}

func defaultConfigPath() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}

	return filepath.Join(usr.HomeDir, "restic_web.conf"), nil
}
