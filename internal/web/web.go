package web

import (
	"os/user"
	"encoding/json"
	"io/ioutil"
	"regexp"
	"strings"
	"path/filepath"
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
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filepath, b, 0600)
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

type AddRepo struct {
	Path    string  `json:"path"`
	Password string `json:"password"`
	Errors  map[string]string `json:"errors"`
}

func (a *AddRepo) Validate() bool {
	a.Errors = make(map[string]string)

	re := regexp.MustCompile(string(filepath.Separator) + ".*")
	matched := re.Match([]byte(a.Path))
	if matched == false {
		a.Errors["Path"] = "Please enter a valid path beginning with " + string(filepath.Separator)
	}

	if strings.TrimSpace(a.Password) == "" {
		a.Errors["Password"] = "Please enter a password"
	}

	return len(a.Errors) == 0
}



//var b []byte
//buf := bytes.NewBuffer(b)
//if err := json.NewEncoder(buf).Encode(addRepo.Errors); err != nil {
//	Verbosef("error encoding response %s\n", err)
//	return
//}
//
//Verbosef("wrote bytes: %s\n", string(buf.Bytes()))
