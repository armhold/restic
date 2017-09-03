package web

import (
	"encoding/base64"
	"net/http"
	"time"
)

// NB: only suitable for localhost usage because cookie contents are not signed/verified

type Flash struct {
	Success string
	Info    string
	Warning string
	Danger  string
}

func SaveFlashToCookie(w http.ResponseWriter, name string, value string) {
	c := &http.Cookie{Name: name, Value: encode([]byte(value))}
	http.SetCookie(w, c)
}

func ParseFlashes(w http.ResponseWriter, r *http.Request) (result Flash, err error) {
	result.Success, err = ReadFlashFromCookie(w, r, "success_flash")
	if err != nil {
		return
	}

	result.Info, err = ReadFlashFromCookie(w, r, "info_flash")
	if err != nil {
		return
	}

	result.Warning, err = ReadFlashFromCookie(w, r, "warn_flash")
	if err != nil {
		return
	}

	result.Danger, err = ReadFlashFromCookie(w, r, "danger_flash")
	if err != nil {
		return
	}

	return
}

func ReadFlashFromCookie(w http.ResponseWriter, r *http.Request, name string) (string, error) {
	c, err := r.Cookie(name)
	if err != nil {
		switch err {
		case http.ErrNoCookie:
			return "", nil
		default:
			return "", err
		}
	}
	b, err := decode(c.Value)
	if err != nil {
		return "", err
	}

	// now that we've read it, tell browser to expire it
	dc := &http.Cookie{Name: name, MaxAge: -1, Expires: time.Unix(1, 0)}
	http.SetCookie(w, dc)
	return string(b), nil
}

func encode(src []byte) string {
	return base64.URLEncoding.EncodeToString(src)
}

func decode(src string) ([]byte, error) {
	return base64.URLEncoding.DecodeString(src)
}
