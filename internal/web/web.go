package web

import (
	"os/user"
	"encoding/json"
	"io/ioutil"
)

type Repo struct {
	Name     string `json:"Name"`     // "local repo"
	Path     string `json:"Path"`     //  "b2:bucket-Name/Path"
	Password string `json:"Password"` // TODO: encrypt?
}

type Config struct {
	Repos []Repo `json:"Repos"`
}

func (c *Config) listRepos() ([]*Repo) {
	var result []*Repo

	return result
}

func (c *Config) Save(filepath string) (error) {
	json, err := json.Marshal(c)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filepath, json, 0600)
}

func LoadConfig(filepath string) (Config, error) {
	var result Config

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

	return usr.HomeDir, nil
}
