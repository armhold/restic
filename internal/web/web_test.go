package web

import (
	"testing"
	"io/ioutil"
	"fmt"
	"reflect"
)

func TestConfig(t *testing.T) {
	configFile, err := ioutil.TempFile("", "Config.json")
	if err != nil {
		t.Error(err)
		return
	}

	repos := []Repo{Repo{Name: "r1", Password: "p1"}, Repo{Name: "r2", Password: "p2"}}

	c := Config{Repos: repos, Filepath: configFile.Name()}

	err = c.Save()
	fmt.Printf("save to : %s\n", configFile.Name())
	if err != nil {
		t.Error(err)
		return
	}

	restored, err := LoadConfig(configFile.Name())
	if err != nil {
		t.Error(err)
		return
	}

	if ! reflect.DeepEqual(c, restored) {
		t.Errorf("%v != %v", c, restored)
	}
}
