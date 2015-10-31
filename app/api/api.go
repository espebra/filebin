package api

import (
	"os/exec"
	"time"
	"net/http"
	"path/filepath"
	"github.com/gorilla/mux"
	"github.com/golang/glog"
	"github.com/espebra/filebin/app/config"
	"github.com/espebra/filebin/app/model"
	"github.com/espebra/filebin/app/output"
)

func triggerNewTagHandler(c string, tag string) error {
	glog.Info("Executing trigger-new-tag: " + c)
	cmd := exec.Command(c, tag)
	err := cmdHandler(cmd)
	return err
}

func triggerUploadedFileHandler(c string, tag string, filename string) error {
	glog.Info("Executing trigger-uploaded-file: " + c)
	cmd := exec.Command(c, tag, filename)
	err := cmdHandler(cmd)
	return err
}

func triggerExpiredTagHandler(c string, tag string) error {
	glog.Info("Executing trigger-expired-tag: " + c)
	cmd := exec.Command(c, tag)
	err := cmdHandler(cmd)
	return err
}

func cmdHandler(cmd *exec.Cmd) error {
	err := cmd.Start()
	if err != nil {
		glog.Error("Trigger command failed: ", err)
	}
	return err
}

func Upload(w http.ResponseWriter, r *http.Request, cfg config.Configuration) {
	var err error

	f := model.File { }
	f.SetFilename(r.Header.Get("filename"))
	err = f.SetTag(r.Header.Get("tag"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest);
		return
	}

	f.TagDir = filepath.Join(cfg.Filedir, f.Tag)
	err = f.EnsureTagDirectoryExists()
	if err != nil {
		glog.Error("Unable to create tag directory", f.TagDir)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError);
		return
	}

	err = f.WriteFile(r.Body)
	if err != nil {
		glog.Info("Unable to write file " + filepath.Join(f.TagDir, f.Filename) + ":", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError);
		return
	}

	err = f.VerifySHA256(r.Header.Get("content-sha256"))
	if err != nil {
		http.Error(w, "Checksum did not match", http.StatusConflict);
		return
	}
	err = f.DetectMIME()
	if err != nil {
		glog.Error("Unable to detect MIME from " + filepath.Join(f.TagDir, f.Filename) + ":", err)
	}
	f.GenerateLinks(cfg.Baseurl)

	f.RemoteAddr = r.RemoteAddr
	f.UserAgent = r.Header.Get("User-Agent")
	f.CreatedAt = time.Now().UTC()
	f.ExpiresAt = time.Now().UTC().Add(24 * 7 * 4 * time.Hour)

	if cfg.TriggerUploadedFile != "" {
		triggerUploadedFileHandler(cfg.TriggerUploadedFile, f.Tag, f.Filename)
	}

	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"

	var status = 201
	output.JSONresponse(w, status, headers, f)
}

func FetchFile(w http.ResponseWriter, r *http.Request, cfg config.Configuration) {
	var err error
	params := mux.Vars(r)
	f := model.File {}
	f.SetFilename(params["filename"])
	err = f.SetTag(params["tag"])
	if err != nil {
	    http.Error(w,"Invalid tag specified. It contains illegal characters or is too short.", 400)
	    return
	}
	
	path := filepath.Join(cfg.Filedir, f.Tag, f.Filename)
	
	w.Header().Set("Cache-Control", "max-age: 60")
	http.ServeFile(w, r, path)
}
