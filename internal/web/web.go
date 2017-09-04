package web

import (
	"os"
	"net/http"
	"fmt"
	"html/template"
)

var (
	// to pass FuncMap, order is important.
	// See: https://stackoverflow.com/questions/17843311/template-and-custom-function-panic-function-not-defined
	templates = template.Must(template.New("").Funcs(Helpers).ParseGlob("internal/web/*.html"))
	WebConfig Config
)

func init() {
}

type Repo struct {
	Name     string `json:"Name"`     // "local repo"
	Path     string `json:"Path"`     //  "b2:bucket-Name/Path"
	Password string `json:"Password"` // TODO: encrypt?
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
	return err
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
