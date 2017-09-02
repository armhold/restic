package web

import (
	"net/http"
	"time"
)

type Page struct {
	Title     string
	StartTime string

	SuccessFlash string
	InfoFlash    string
	WarningFlash string
	DangerFlash  string
}

func PageFromRequest(title string, w http.ResponseWriter, r *http.Request) (Page, error) {
	result := Page{Title: title, StartTime: time.Now().Format(time.Stamp)}

	flash, err := ReadFlashFromCookie(w, r, "success_flash")
	if err != nil {
		return result, err
	}
	result.SuccessFlash = flash

	flash, err = ReadFlashFromCookie(w, r, "info_flash")
	if err != nil {
		return result, err
	}
	result.InfoFlash = flash

	flash, err = ReadFlashFromCookie(w, r, "warn_flash")
	if err != nil {
		return result, err
	}
	result.WarningFlash = flash

	flash, err = ReadFlashFromCookie(w, r, "danger_flash")
	if err != nil {
		return result, err
	}
	result.DangerFlash = flash

	return result, err
}
