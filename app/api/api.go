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

func Upload(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	var err error
	f := model.ExtendedFile { }

	// Extract the tag from the request
	if (r.Header.Get("tag") == "") {
		err = f.GenerateTagID()
	} else {
		err = f.SetTagID(r.Header.Get("tag"))
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest);
		glog.Info(err)
		return
	}
	f.SetTagDir(cfg.Filedir)

	// Write the request body to a temporary file
	err = f.WriteTempfile(r.Body, cfg.Tempdir)
	if err != nil {
		glog.Error("Unable to write tempfile: ", err)

		// Clean up by removing the tempfile
		f.ClearTemp()

		http.Error(w, "Internal Server Error", http.StatusInternalServerError);
		return
	}

	// Do not accept files that are 0 bytes
	if f.Bytes == 0 {
		// Clean up by removing the tempfile
		f.ClearTemp()

		http.Error(w, "No content. The file size must be more than " +
			"0 bytes.", http.StatusBadRequest);
		return
	}

	// Calculate and verify the checksum
	err = f.VerifySHA256(r.Header.Get("content-sha256"))
	if err != nil {
		http.Error(w, "Checksum did not match", http.StatusConflict);
		return
	}

	// Create the tag directory if it does not exist
	err = f.EnsureTagDirectoryExists()
	if err != nil {
		glog.Error("Unable to create tag directory: ", f.TagDir)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError);
		return
	}

	f.CalculateExpiration(cfg.Expiration)
	expired, err := f.IsExpired(cfg.Expiration)
	if err != nil {
		http.Error(w,"Internal server error", 500)
		return
	}
	if expired {
		http.Error(w,"This tag has expired.", 410)
		return
	}

	// Extract the filename from the request
	if (r.Header.Get("filename") == "") {
		glog.Info("Using the checksum " + f.Checksum + " as the " +
			"filename")
		f.SetFilename(f.Checksum)
	} else {
		err = f.SetFilename(r.Header.Get("filename"))
		if err != nil {
			glog.Info(err)
			http.Error(w, "Invalid filename specified. It contains illegal characters or is too short.",
				http.StatusBadRequest);
			return
		}
	}

	// Promote file from tempdir to the published tagdir
	f.Publish()

	// Clean up by removing the tempfile
	f.ClearTemp()

	err = f.DetectMIME()
	if err != nil {
		glog.Error("Unable to detect MIME: ", err)
	}

	err = f.Info()
	if err != nil {
		http.Error(w,"Internal Server Error", 500)
		return
	}

	f.GenerateLinks(cfg.Baseurl)
	f.RemoteAddr = r.RemoteAddr
	f.UserAgent = r.Header.Get("User-Agent")
	f.CreatedAt = time.Now().UTC()
	//f.ExpiresAt = time.Now().UTC().Add(24 * 7 * 4 * time.Hour)

	if cfg.TriggerUploadedFile != "" {
		triggerUploadedFileHandler(cfg.TriggerUploadedFile, f.TagID, f.Filename)
	}

	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"

	var status = 201
	output.JSONresponse(w, status, headers, f)
}

func FetchFile(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	var err error
	params := mux.Vars(r)
	f := model.File {}
	f.SetFilename(params["filename"])
	if err != nil {
		http.Error(w,"Invalid filename specified. It contains illegal characters or is too short.", 400)
		return
	}
	err = f.SetTagID(params["tag"])
	if err != nil {
		http.Error(w,"Invalid tag specified. It contains illegal characters or is too short.", 400)
		return
	}
	f.SetTagDir(cfg.Filedir)

	f.CalculateExpiration(cfg.Expiration)
	expired, err := f.IsExpired(cfg.Expiration)
	if err != nil {
		http.Error(w,"Internal server error", 500)
		return
	}
	if expired {
		http.Error(w,"This tag has expired.", 410)
		return
	}
	
	path := filepath.Join(f.TagDir, f.Filename)
	
	w.Header().Set("Cache-Control", "max-age=1")
	http.ServeFile(w, r, path)
}

func DeleteFile(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	var err error
	params := mux.Vars(r)
	f := model.File {}
	f.SetFilename(params["filename"])
	if err != nil {
		http.Error(w,"Invalid filename specified. It contains illegal characters or is too short.", 400)
		return
	}
	err = f.SetTagID(params["tag"])
	if err != nil {
		http.Error(w,"Invalid tag specified. It contains illegal characters or is too short.", 400)
		return
	}
	f.SetTagDir(cfg.Filedir)

	if f.Exists() == false {
		http.Error(w,"File Not Found", 404)
		return
	}

	f.CalculateExpiration(cfg.Expiration)
	expired, err := f.IsExpired(cfg.Expiration)
	if err != nil {
		http.Error(w,"Internal server error", 500)
		return
	}
	if expired {
		http.Error(w,"This tag has expired.", 410)
		return
	}

	f.GenerateLinks(cfg.Baseurl)
	err = f.DetectMIME()
	if err != nil {
		glog.Error("Unable to detect MIME: ", err)
	}

	err = f.Info()
	if err != nil {
		http.Error(w,"Internal Server Error", 500)
		return
	}

	err = f.Remove()
 	if err != nil {
		glog.Error("Unable to delete file: ", err)
		http.Error(w,"Internal Server Error", 500)
		return
	}

	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"

	var status = 200
	output.JSONresponse(w, status, headers, f)
	return
}

func FetchTag(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	var err error
	params := mux.Vars(r)
	t := model.ExtendedTag {}
	err = t.SetTagID(params["tag"])
	if err != nil {
		http.Error(w, "Invalid tag", 400)
		return
	}

	t.SetTagDir(cfg.Filedir)
	t.CalculateExpiration(cfg.Expiration)
	if t.Exists() {
		expired, err := t.IsExpired(cfg.Expiration)
		if err != nil {
			http.Error(w,"Internal server error", 500)
			return
		}
		if expired {
			http.Error(w,"This tag has expired.", 410)
			return
		}

		err = t.Info()
		if err != nil {
			http.Error(w, "Internal Server Error", 500)
			return
		}

		err = t.List(cfg.Baseurl)
		if err != nil {
			http.Error(w,"Some error.", 404)
			return
		}
	}

	//t.GenerateLinks(cfg.Baseurl)

	headers := make(map[string]string)
	headers["Cache-Control"] = "max-age=1"

	var status = 200

	if (r.Header.Get("Content-Type") == "application/json") {
		headers["Content-Type"] = "application/json"
		output.JSONresponse(w, status, headers, t)
	} else {
		output.HTMLresponse(w, "viewtag", status, headers, t, ctx)
	}
}

func ViewIndex(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	t := model.Tag {}
	err := t.GenerateTagID()
	if err != nil {
		http.Error(w, "Internal Server Error", 500)
		return
	}

	headers := make(map[string]string)
	headers["Cache-Control"] = "max-age=0"
	headers["Location"] = ctx.Baseurl + "/" + t.TagID
	var status = 302
	output.JSONresponse(w, status, headers, t)
}

func ViewAPI(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	t := model.Tag {}
	headers := make(map[string]string)
	headers["Cache-Control"] = "max-age=1"
	var status = 200
	output.HTMLresponse(w, "api", status, headers, t, ctx)
}

func ViewDoc(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	t := model.Tag {}
	headers := make(map[string]string)
	headers["Cache-Control"] = "max-age=1"
	var status = 200
	output.HTMLresponse(w, "doc", status, headers, t, ctx)
}
