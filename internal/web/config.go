package web

import (
	"io/ioutil"
	"path/filepath"
	"errors"
	"encoding/json"
	"os"
	"fmt"
	"os/user"
)

// Config contains all the persistent configuration for the repos managed by the web interface
type Config struct {
	Repos    []Repo `json:"Repos"`
	Filepath string `json:"-"`
}

func (c *Config) listRepos() ([]*Repo) {
	var result []*Repo

	return result
}

func (c *Config) Save() (error) {
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


