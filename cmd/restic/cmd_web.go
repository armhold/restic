package main

import (
	"github.com/spf13/cobra"
	"net/http"
	"time"
	"html/template"
	"os"
	"fmt"
	"regexp"
	"strings"
	"encoding/json"
)

var cmdWeb = &cobra.Command{
	Use:   "web [flags]",
	Short: "start the restic web server",
	Long: `
The "web" command starts up a web server for running backups, restores, etc.
`,
	DisableAutoGenTag: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWeb(webOptions, globalOptions, args)
	},
}

// WebOptions collects all options for the web command.
type WebOptions struct {
	port int
	bindHost string
}

var startTime = time.Now().Format(time.Stamp)

var webOptions WebOptions

var templates = template.Must(template.ParseGlob("templates/*.html"))


type Page struct {
	Title string
	StartTime string
}


func init() {
	cmdRoot.AddCommand(cmdWeb)

	flags := cmdWeb.Flags()
	flags.StringVarP(&webOptions.bindHost, "host", "H", "localhost", "hostname to bind to")
	flags.IntVar(&webOptions.port,  "port", 8080, "port to bind to")
}



func runWeb(opts WebOptions, gopts GlobalOptions, args []string) error {
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/addrepo", addRepoHandler)

	// static assets
	fs := JustFilesFilesystem{http.Dir("assets")}
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(fs)))

	addr := fmt.Sprintf("%s:%d", opts.bindHost, opts.port)

	Verbosef("binding to %s\n", addr)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		Exitf(1, err.Error())
	}

	return nil
}


func rootHandler(w http.ResponseWriter, r *http.Request) {
	Verbosef("rootHandler")
	Verbosef("path: %q\n", r.URL.Path)

	// The "/" pattern matches everything, so we need to check that we're at the root here.
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	p := &Page{Title: "Root Page", StartTime: startTime}
	if err := templates.ExecuteTemplate(w, "index.html", p) ; err != nil {
		Verbosef(err.Error())
	}
}

type AddRepo struct {
	Path    string
	Password string
	Errors  map[string]string
}

func (a *AddRepo) Validate() bool {
	a.Errors = make(map[string]string)

	re := regexp.MustCompile("/.*")
	matched := re.Match([]byte(a.Path))
	if matched == false {
		a.Errors["Path"] = "Please enter a valid path"
	}

	if strings.TrimSpace(a.Password) == "" {
		Verbosef("no password specified\n")
		a.Errors["Password"] = "Please enter a password"
	}

	return len(a.Errors) == 0
}





func addRepoHandler(w http.ResponseWriter, r *http.Request) {
	Verbosef("addRepoHandler\n")
	w.Header().Set("Content-Type", "application/json")


	err := r.ParseForm()
	if err != nil {
		Verbosef("error parsing form: %s\n", err.Error())
		return
	}
	Verbosef("addRepoHandler %v\n", r.Form)

	addRepo := &AddRepo{
		Path: r.FormValue("path"),
		Password: r.FormValue("password"),
	}

	if addRepo.Validate() == false {
		w.WriteHeader(http.StatusBadRequest)
	} else {
		w.WriteHeader(http.StatusOK)
		//http.Redirect(w, r, "/", http.StatusSeeOther)
	}

	if err := json.NewEncoder(w).Encode(addRepo); err != nil {
		Verbosef("error encoding response %s\n", err)
		return
	}
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
