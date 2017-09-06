package web

import (
	"net/http"
	"os"
)

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
