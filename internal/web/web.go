package web

import (
	"os"
	"os/user"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"fmt"
	"html/template"
	"time"
	"path/filepath"
)

var (
	startTime = time.Now().Format(time.Stamp)
	templates = template.Must(template.ParseGlob("internal/web/*.html"))
	WebConfig  Config
)


func init() {
}

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

func LoadConfigFromDefault() (Config, error) {
	path, err := defaultConfigPath()
	if err != nil {
		return Config{}, err
	}

	result, err := LoadConfig(path)
	if os.IsNotExist(err) {
		return SaveDefaultConfig()
	}

	return result, err
}

func SaveDefaultConfig() (Config, error) {
	result := Config{}
	path, err := defaultConfigPath()
	if err != nil {
		return result, err
	}

	fmt.Printf("saving default config to %s\n", path)

	err = result.Save(path)
	return result, err
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

	return  filepath.Join(usr.HomeDir, "restic_web.conf"), nil
}

//var b []byte
//buf := bytes.NewBuffer(b)
//if err := json.NewEncoder(buf).Encode(addRepo.Errors); err != nil {
//	fmt.Printf("error encoding response %s\n", err)
//	return
//}
//
//fmt.Printf("wrote bytes: %s\n", string(buf.Bytes()))

func RunWeb(bindHost string, bindPort int) error {

	c, err := LoadConfigFromDefault()
	if err != nil {
		return err
	}

	WebConfig = c

	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/addrepo", AddRepoAjaxHandler)

	// static assets
	fs := JustFilesFilesystem{http.Dir("assets")}
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(fs)))

	addr := fmt.Sprintf("%s:%d", bindHost, bindPort)

	fmt.Printf("binding to %s\n", addr)
	err = http.ListenAndServe(addr, nil)
	if err != nil {
		return err
	}

	return nil
}

// FileSystem that prevents directory listing.
//
// http://grokbase.com/t/gg/golang-nuts/12a9yhgr64/go-nuts-disable-directory-listing-with-http-fileserver/oldest#201210095mknmkj366el5oujntxmxybfga
//
type JustFilesFilesystem struct {
	Fs http.FileSystem
}

func (fs JustFilesFilesystem) Open(name string) (http.File, error) {
	f, err := fs.Fs.Open(name)
	if err != nil {
		return nil, err
	}

	stat, err := f.Stat()
	if stat.IsDir() {
		return nil, os.ErrNotExist
	}

	//	return neuteredReaddirFile{f}, nil
	return f, nil
}

//type neuteredReaddirFile struct {
//	http.File
//}
//
//func (f neuteredReaddirFile) Readdir(count int) ([]os.FileInfo, error) {
//	return nil, nil
//}
//
