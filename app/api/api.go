package api

import (
	"os/exec"
	"time"
	"strconv"
	"net/http"
        "math/rand"
	"path/filepath"
	"regexp"
	"fmt"
	"net/url"

	"github.com/gorilla/mux"
	"github.com/espebra/filebin/app/config"
	"github.com/espebra/filebin/app/model"
	"github.com/espebra/filebin/app/output"
)

func isWorkaroundNeeded(useragent string) bool {
	matched, err := regexp.MatchString("(iPhone|iPad|iPod)", useragent)
	if err != nil {
		fmt.Println(err)
	}
	return matched
}

func triggerNewTagHandler(c string, tag string) error {
	cmd := exec.Command(c, tag)
	err := cmdHandler(cmd)
	return err
}

func triggerUploadedFileHandler(c string, tag string, filename string) error {
	cmd := exec.Command(c, tag, filename)
	err := cmdHandler(cmd)
	return err
}

func triggerExpiredTagHandler(c string, tag string) error {
	cmd := exec.Command(c, tag)
	err := cmdHandler(cmd)
	return err
}

func cmdHandler(cmd *exec.Cmd) error {
	err := cmd.Start()
	return err
}

func randomString(n int) string {
        var letters = []rune("abcdefghijklmnopqrstuvwxyz0123456789")
        b := make([]rune, n)
        for i := range b {
                b[i] = letters[rand.Intn(len(letters))]
        }
        return string(b)
}

func Upload(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	var err error
	f := model.File { }
	f.RemoteAddr = r.RemoteAddr
	f.UserAgent = r.Header.Get("User-Agent")

	// Extract the tag from the request
	if (r.Header.Get("tag") == "") {
		tag := randomString(cfg.DefaultTagLength)
		err = f.SetTag(tag)
		ctx.Log.Println("Tag generated: " + f.Tag)
	} else {
		tag := r.Header.Get("tag")
		err = f.SetTag(tag)
		ctx.Log.Println("Tag specified: " + tag)
	}
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest);
		return
	}
	f.SetTagDir(cfg.Filedir)
	ctx.Log.Println("Tag directory: " + f.TagDir)

	// Write the request body to a temporary file
	err = f.WriteTempfile(r.Body, cfg.Tempdir)
	if err != nil {
		ctx.Log.Println("Unable to write tempfile: ", err)

		// Clean up by removing the tempfile
		f.ClearTemp()

		http.Error(w, "Internal Server Error", http.StatusInternalServerError);
		return
	}
	ctx.Log.Println("Tempfile: " + f.Tempfile)
	ctx.Log.Println("Tempfile size: " + strconv.FormatInt(f.Bytes, 10) + " bytes")

	// Do not accept files that are 0 bytes
	if f.Bytes == 0 {
		ctx.Log.Println("Empty files are not allowed. Aborting.")

		// Clean up by removing the tempfile
		f.ClearTemp()

		http.Error(w, "No content. The file size must be more than " +
			"0 bytes.", http.StatusBadRequest);
		return
	}

	// Calculate and verify the checksum
	checksum := r.Header.Get("content-sha256")
	if checksum != "" {
		ctx.Log.Println("Checksum specified: " + checksum)
	}
	err = f.VerifySHA256(checksum)
	ctx.Log.Println("Checksum calculated: " + f.Checksum)
	if err != nil {
		ctx.Log.Println("The specified checksum did not match")
		http.Error(w, "Checksum did not match", http.StatusConflict);
		return
	}

	// Trigger new tag
	t := model.Tag{}
	t.SetTag(f.Tag)
	t.SetTagDir(cfg.Filedir)
	if !t.TagDirExists() {
		if cfg.TriggerNewTag != "" {
			ctx.Log.Println("Executing trigger: New tag")
			triggerNewTagHandler(cfg.TriggerNewTag, f.Tag)
		}
	}

	// Create the tag directory if it does not exist
	err = f.EnsureTagDirectoryExists()
	if err != nil {
		ctx.Log.Println("Unable to create tag directory: ", f.TagDir)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError);
		return
	}

	t.CalculateExpiration(cfg.Expiration)
	expired, err := t.IsExpired(cfg.Expiration)
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w,"Internal server error", 500)
		return
	}
	if expired {
		ctx.Log.Println("The tag has expired. Aborting.")
		http.Error(w,"This tag has expired.", 410)
		return
	}

	// Extract the filename from the request
	fname := r.Header.Get("filename")
	if (fname == "") {
		ctx.Log.Println("Filename generated: " + f.Checksum)
		f.SetFilename(f.Checksum)
	} else {
		ctx.Log.Println("Filename specified: " + fname)
		err = f.SetFilename(fname)
		if err != nil {
			ctx.Log.Println(err)
			http.Error(w, "Invalid filename specified. It contains illegal characters or is too short.",
				http.StatusBadRequest);
			return
		}
	}

	if fname != f.Filename {
		ctx.Log.Println("Filename sanitized: " + f.Filename)
	}

	err = f.DetectMIME()
	if err != nil {
		ctx.Log.Println("Unable to detect MIME: ", err)
	} else {
		ctx.Log.Println("MIME detected: " + f.MIME)
	}

	ctx.Log.Println("Media type: " + f.MediaType())
        if f.MediaType() == "image" {
                err = f.ParseExif()
                if err != nil {
                        ctx.Log.Println(err)
		}

                err = f.ExtractDateTime()
                if err != nil {
                        ctx.Log.Println(err)
		}

		// iOS devices provide only one filename even when uploading
		// multiple images. Providing some workaround for this below.
		// XXX: Refactoring needed.
		if isWorkaroundNeeded(f.UserAgent) && !f.DateTime.IsZero() {
			var fname string
			dt := f.DateTime.Format("060102-150405")

			// List of filenames to modify
			if (f.Filename == "image.jpeg") {
				fname = "img-" + dt + ".jpeg"
			}
			if (f.Filename == "image.gif") {
				fname = "img-" + dt + ".gif"
			}
			if (f.Filename == "image.png") {
				fname = "img-" + dt + ".png"
			}

			if fname != "" {
				ctx.Log.Println("Filename workaround triggered")
				ctx.Log.Println("Filename modified: " + fname)
				err = f.SetFilename(fname)
				if err != nil {
					ctx.Log.Println(err)
				}
			}
		}

		//err = f.GenerateThumbnail()
		//if err != nil {
		//	ctx.Log.Println(err)
		//}

		extra := make(map[string]string)
		if !f.DateTime.IsZero() {
			extra["DateTime"] = f.DateTime.String()
		}
		f.Extra = extra
        }


	// Promote file from tempdir to the published tagdir
	f.Publish()

	// Clean up by removing the tempfile
	f.ClearTemp()

	err = f.StatInfo()
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w,"Internal Server Error", 500)
		return
	}

	f.GenerateLinks(cfg.Baseurl)
	f.CreatedAt = time.Now().UTC()
	//f.ExpiresAt = time.Now().UTC().Add(24 * 7 * 4 * time.Hour)

	if cfg.TriggerUploadedFile != "" {
		ctx.Log.Println("Executing trigger: Uploaded file")
		triggerUploadedFileHandler(cfg.TriggerUploadedFile, f.Tag, f.Filename)
	}

	ctx.WorkQueue <- f

	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"

	var status = 201
	output.JSONresponse(w, status, headers, f, ctx)
}

func FetchFile(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	var err error

	// Query parameters
	u, err := url.Parse(r.RequestURI)
	if err != nil {
		ctx.Log.Println(err)
	}

	queryParams, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		ctx.Log.Println(err)
	}

	// Request headers
	params := mux.Vars(r)

	f := model.File {}
	f.SetFilename(params["filename"])
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w,"Invalid filename specified. It contains illegal characters or is too short.", 400)
		return
	}
	err = f.SetTag(params["tag"])
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w,"Invalid tag specified. It contains illegal characters or is too short.", 400)
		return
	}
	f.SetTagDir(cfg.Filedir)

	t := model.Tag { }
	t.SetTag(f.Tag)
	t.SetTagDir(cfg.Filedir)
	t.CalculateExpiration(cfg.Expiration)
	expired, err := t.IsExpired(cfg.Expiration)
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w,"Internal server error", 500)
		return
	}
	if expired {
		ctx.Log.Println("Expired: " + t.ExpirationReadable)
		http.Error(w,"This tag has expired.", 410)
		return
	}
	
	// Default path
	path := filepath.Join(f.TagDir, f.Filename)

	for param, values := range queryParams {
		if param == "size" && values[0] == "thumbnail" {
			if f.ThumbnailExists() {
				path = f.ThumbnailPath()
			} else {
				http.Error(w,"Thumbnail not found", 404)
				return
			}
		}
	}
	
	w.Header().Set("Cache-Control", "max-age=1")
	http.ServeFile(w, r, path)
}

func DeleteFile(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	var err error
	params := mux.Vars(r)
	f := model.File {}
	f.SetFilename(params["filename"])
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w,"Invalid filename specified. It contains illegal characters or is too short.", 400)
		return
	}
	err = f.SetTag(params["tag"])
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w,"Invalid tag specified. It contains illegal characters or is too short.", 400)
		return
	}
	f.SetTagDir(cfg.Filedir)

	if f.Exists() == false {
		ctx.Log.Println("The file does not exist.")
		http.Error(w,"File Not Found", 404)
		return
	}

	t := model.Tag { }
	t.SetTag(f.Tag)
	t.SetTagDir(cfg.Filedir)
	t.CalculateExpiration(cfg.Expiration)
	expired, err := t.IsExpired(cfg.Expiration)
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w,"Internal server error", 500)
		return
	}
	if expired {
		ctx.Log.Println("Expired: " + t.ExpirationReadable)
		http.Error(w,"This tag has expired.", 410)
		return
	}

	f.GenerateLinks(cfg.Baseurl)
	err = f.DetectMIME()
	if err != nil {
		ctx.Log.Println("Unable to detect MIME: ", err)
	}

	err = f.StatInfo()
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w,"Internal Server Error", 500)
		return
	}

	err = f.Remove()
 	if err != nil {
		ctx.Log.Println("Unable to remove file: ", err)
		http.Error(w,"Internal Server Error", 500)
		return
	}

	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"

	var status = 200
	output.JSONresponse(w, status, headers, f, ctx)
	return
}

func FetchTag(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	var err error
	params := mux.Vars(r)
	t := model.Tag {}
	err = t.SetTag(params["tag"])
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w, "Invalid tag", 400)
		return
	}

	t.SetTagDir(cfg.Filedir)
	t.CalculateExpiration(cfg.Expiration)
	if t.TagDirExists() {
		expired, err := t.IsExpired(cfg.Expiration)
		if err != nil {
			ctx.Log.Println(err)
			http.Error(w,"Internal server error", 500)
			return
		}
		if expired {
			ctx.Log.Println("Expired: " + t.ExpirationReadable)
			http.Error(w,"This tag has expired.", 410)
			return
		}

		err = t.StatInfo()
		if err != nil {
			ctx.Log.Println(err)
			http.Error(w, "Internal Server Error", 500)
			return
		}

		err = t.List(cfg.Baseurl)
		if err != nil {
			ctx.Log.Println(err)
			http.Error(w,"Error reading the tag contents.", 404)
			return
		}
	}

	//t.GenerateLinks(cfg.Baseurl)

	headers := make(map[string]string)
	headers["Cache-Control"] = "max-age=1"

	var status = 200

	if (r.Header.Get("Content-Type") == "application/json") {
		headers["Content-Type"] = "application/json"
		output.JSONresponse(w, status, headers, t, ctx)
	} else {
		output.HTMLresponse(w, "viewtag", status, headers, t, ctx)
	}
}

func ViewIndex(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	t := model.Tag {}
	tag := randomString(cfg.DefaultTagLength)
	err := t.SetTag(tag)
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}
	ctx.Log.Println("Tag generated: " + t.Tag)

	headers := make(map[string]string)
	headers["Cache-Control"] = "max-age=0"
	headers["Location"] = ctx.Baseurl + "/" + t.Tag
	var status = 302
	output.JSONresponse(w, status, headers, t, ctx)
}

//func ViewAPI(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
//	t := model.Tag {}
//	headers := make(map[string]string)
//	headers["Cache-Control"] = "max-age=1"
//	var status = 200
//	output.HTMLresponse(w, "api", status, headers, t, ctx)
//}
//
//func ViewDoc(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
//	t := model.Tag {}
//	headers := make(map[string]string)
//	headers["Cache-Control"] = "max-age=1"
//	var status = 200
//	output.HTMLresponse(w, "doc", status, headers, t, ctx)
//}
