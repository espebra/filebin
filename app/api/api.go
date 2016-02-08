package api

import (
	"fmt"
	"syscall"
	"math/rand"
	"net/http"
	"net/url"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

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

func triggerUploadFileHandler(c string, tag string, filename string) error {
	cmd := exec.Command(c, tag, filename)
	err := cmdHandler(cmd)
	return err
}

func triggerDownloadTagHandler(c string, tag string) error {
	cmd := exec.Command(c, tag)
	err := cmdHandler(cmd)
	return err
}

func triggerDownloadFileHandler(c string, tag string, filename string) error {
	cmd := exec.Command(c, tag, filename)
	err := cmdHandler(cmd)
	return err
}

func triggerDeleteTagHandler(c string, tag string) error {
	cmd := exec.Command(c, tag)
	err := cmdHandler(cmd)
	return err
}

func triggerDeleteFileHandler(c string, tag string, filename string) error {
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
	f := model.File{}
	f.RemoteAddr = r.RemoteAddr
	f.UserAgent = r.Header.Get("User-Agent")

	// Extract the tag from the request
	if r.Header.Get("tag") == "" {
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
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	f.SetTagDir(cfg.Filedir)
	ctx.Log.Println("Tag directory: " + f.TagDir)

	contentLength, err := strconv.ParseUint(r.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		ctx.Log.Println(err)
	}
	ctx.Log.Printf("Specified content length: %d bytes", contentLength)

	// Read the amounts of bytes free in the filedir directory
	var stat syscall.Statfs_t
	syscall.Statfs(cfg.Filedir, &stat)
	freeBytes := stat.Bavail * uint64(stat.Bsize)
	ctx.Log.Printf("Free storage: %d bytes", freeBytes)

	if contentLength >= freeBytes {
		ctx.Log.Println("Not enough free disk space for the specified content-length. Trying to abort here.")

		// This will not work as expected since clients (usually) don't care about
		// the response until the request delivery is complete.
		http.Error(w, "Request Entity Too Large", 413)
		return
	}

	// Write the request body to a temporary file
	err = f.WriteTempfile(r.Body, cfg.Tempdir)
	if err != nil {
		ctx.Log.Println("Unable to write tempfile: ", err)

		// Clean up by removing the tempfile
		f.ClearTemp()

		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	ctx.Log.Println("Tempfile: " + f.Tempfile)
	ctx.Log.Println("Tempfile size: " + strconv.FormatInt(f.Bytes, 10) + " bytes")

	// Do not accept files that are 0 bytes
	if f.Bytes == 0 {
		ctx.Log.Println("Empty files are not allowed. Aborting.")

		// Clean up by removing the tempfile
		f.ClearTemp()

		http.Error(w, "No content. The file size must be more than "+
			"0 bytes.", http.StatusBadRequest)
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
		http.Error(w, "Checksum did not match", http.StatusConflict)
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
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	t.CalculateExpiration(cfg.Expiration)
	expired, err := t.IsExpired(cfg.Expiration)
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w, "Internal server error", 500)
		return
	}
	if expired {
		ctx.Log.Println("The tag has expired. Aborting.")
		http.Error(w, "This tag has expired.", 410)
		return
	}

	// Extract the filename from the request
	fname := r.Header.Get("filename")
	if fname == "" {
		ctx.Log.Println("Filename generated: " + f.Checksum)
		f.SetFilename(f.Checksum)
	} else {
		ctx.Log.Println("Filename specified: " + fname)
		err = f.SetFilename(fname)
		if err != nil {
			ctx.Log.Println(err)
			http.Error(w, "Invalid filename specified. It contains illegal characters or is too short.",
				http.StatusBadRequest)
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

		// iOS devices provide only one filename even when uploading
		// multiple images. Providing some workaround for this below.
		// XXX: Refactoring needed.
		if isWorkaroundNeeded(f.UserAgent) && !f.DateTime.IsZero() {
			var fname string
			dt := f.DateTime.Format("060102-150405")

			// List of filenames to modify
			if f.Filename == "image.jpeg" {
				fname = "img-" + dt + ".jpeg"
			}
			if f.Filename == "image.gif" {
				fname = "img-" + dt + ".gif"
			}
			if f.Filename == "image.png" {
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

		//extra := make(map[string]string)
		//if !f.DateTime.IsZero() {
		//	extra["DateTime"] = f.DateTime.String()
		//}
		//f.Extra = extra
	}

	// Promote file from tempdir to the published tagdir
	f.Publish()

	// Clean up by removing the tempfile
	f.ClearTemp()

	err = f.StatInfo()
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	f.GenerateLinks(cfg.Baseurl)
	f.CreatedAt = time.Now().UTC()
	//f.ExpiresAt = time.Now().UTC().Add(24 * 7 * 4 * time.Hour)

	if cfg.TriggerUploadFile != "" {
		ctx.Log.Println("Executing trigger: Uploaded file")
		triggerUploadFileHandler(cfg.TriggerUploadFile, f.Tag, f.Filename)
	}

	// Purging any old content
	if cfg.CacheInvalidation {
		if err := f.Purge(); err != nil {
			ctx.Log.Println(err)
		}
	}

	ctx.WorkQueue <- f

	w.Header().Set("Content-Type", "application/json")

	var status = 201
	output.JSONresponse(w, status, f, ctx)
}

func FetchFile(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
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

	f := model.File{}
	f.SetFilename(params["filename"])
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w, "Invalid filename specified. It contains illegal characters or is too short.", 400)
		return
	}
	err = f.SetTag(params["tag"])
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w, "Invalid tag specified. It contains illegal characters or is too short.", 400)
		return
	}
	f.SetTagDir(cfg.Filedir)

	t := model.Tag{}
	t.SetTag(f.Tag)
	t.SetTagDir(cfg.Filedir)
	t.CalculateExpiration(cfg.Expiration)
	expired, err := t.IsExpired(cfg.Expiration)
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}
	if expired {
		ctx.Log.Println("Expired: " + t.ExpirationReadable)
		http.Error(w, "This tag has expired.", 410)
		return
	}

	// Default path
	path := filepath.Join(f.TagDir, f.Filename)

	err = f.DetectMIME()
	if err != nil {
		ctx.Log.Println("Unable to detect MIME: ", err)
	} else {
		ctx.Log.Println("MIME detected: " + f.MIME)
	}

	if f.MediaType() == "image" {
		width, _ := strconv.Atoi(queryParams.Get("width"))
		height, _ := strconv.Atoi(queryParams.Get("height"))
		if (width > 0) || (height > 0) {
			ctx.Log.Println("Size requested: " +
				strconv.Itoa(width) + "x" +
				strconv.Itoa(height) + " px")
			if f.ImageExists(width, height) {
				path = f.ImagePath(width, height)
			} else {
				http.Error(w, "Image not found", 404)
				return
			}
		}
	}

	if cfg.TriggerDownloadFile != "" {
		ctx.Log.Println("Executing trigger: Download file")
		triggerDownloadFileHandler(cfg.TriggerDownloadFile, f.Tag, f.Filename)
	}

	w.Header().Set("Vary", "Content-Type")
	w.Header().Set("Cache-Control", "s-maxage=3600")
	http.ServeFile(w, r, path)
}

func DeleteTag(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	var err error
	params := mux.Vars(r)

	t := model.Tag{}

	err = t.SetTag(params["tag"])
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w, "Invalid tag", 400)
		return
	}
	t.SetTagDir(cfg.Filedir)

	// Tag does not exist
	if t.TagDirExists() == false {
		http.Error(w, "Tag Not Found", 404)
		return
	}

	t.List(cfg.Baseurl)
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	if cfg.TriggerDeleteTag != "" {
		ctx.Log.Println("Executing trigger: Delete tag")
		triggerDeleteTagHandler(cfg.TriggerDeleteTag, t.Tag)
	}

	// Tag exists, so let's remove it.
	err = t.Remove()
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	// Verify that the tag directory is removed before sending the response.
	if t.TagDirExists() == true {
		// Failsafe. This should not happen.
		ctx.Log.Println("Failed to delete the tag. The tag dir still exists.")
		http.Error(w, "Internal Server Error", 500)
		return
	}

	// Purging any old content
	if cfg.CacheInvalidation {
		for _, f := range t.Files {
			if err := f.Purge(); err != nil {
				ctx.Log.Println(err)
			}
		}
	}

	ctx.Log.Println("Tag deleted successfully.")
	http.Error(w, "Tag Deleted Successfully", 200)
	return

}

func DeleteFile(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	var err error
	params := mux.Vars(r)
	f := model.File{}
	f.SetFilename(params["filename"])
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w, "Invalid filename specified. It contains illegal characters or is too short.", 400)
		return
	}
	err = f.SetTag(params["tag"])
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w, "Invalid tag specified. It contains illegal characters or is too short.", 400)
		return
	}
	f.SetTagDir(cfg.Filedir)

	if f.Exists() == false {
		ctx.Log.Println("The file does not exist.")
		http.Error(w, "File Not Found", 404)
		return
	}

	t := model.Tag{}
	t.SetTag(f.Tag)
	t.SetTagDir(cfg.Filedir)
	t.CalculateExpiration(cfg.Expiration)
	expired, err := t.IsExpired(cfg.Expiration)
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w, "Internal server error", 500)
		return
	}
	if expired {
		ctx.Log.Println("Expired: " + t.ExpirationReadable)
		http.Error(w, "This tag has expired.", 410)
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
		http.Error(w, "Internal Server Error", 500)
		return
	}

	if cfg.TriggerDeleteFile != "" {
		ctx.Log.Println("Executing trigger: Delete file")
		triggerDeleteFileHandler(cfg.TriggerDeleteFile, f.Tag, f.Filename)
	}

	err = f.Remove()
	if err != nil {
		ctx.Log.Println("Unable to remove file: ", err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	// Purging any old content
	if cfg.CacheInvalidation {
		if err := f.Purge(); err != nil {
			ctx.Log.Println(err)
		}
	}

	w.Header().Set("Content-Type", "application/json")

	var status = 200
	output.JSONresponse(w, status, f, ctx)
	return
}

func FetchAlbum(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	t := model.Tag{}
	params := mux.Vars(r)
	err := t.SetTag(params["tag"])
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
			http.Error(w, "Internal server error", 500)
			return
		}
		if expired {
			ctx.Log.Println("Expired: " + t.ExpirationReadable)
			http.Error(w, "This tag has expired.", 410)
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
			http.Error(w, "Error reading the tag contents.", 404)
			return
		}
	} else {
		// The tag does not exist
		http.Error(w, "Not found", 404)
		return
	}

	w.Header().Set("Cache-Control", "s-maxage=3600")

	var status = 200
	output.HTMLresponse(w, "viewalbum", status, t, ctx)
	return
}

func FetchTag(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	params := mux.Vars(r)
	t := model.Tag{}
	err := t.SetTag(params["tag"])
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
			http.Error(w, "Internal server error", 500)
			return
		}
		if expired {
			ctx.Log.Println("Expired: " + t.ExpirationReadable)
			http.Error(w, "This tag has expired.", 410)
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
			http.Error(w, "Error reading the tag contents.", 404)
			return
		}
	}

	w.Header().Set("Vary", "Content-Type")
	w.Header().Set("Cache-Control", "s-maxage=3600")

	var status = 200

	if r.Header.Get("Content-Type") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		output.JSONresponse(w, status, t, ctx)
		return
	} else {
		if len(t.Files) == 0 {
			output.HTMLresponse(w, "newtag", status, t, ctx)
		} else {
			output.HTMLresponse(w, "viewtag", status, t, ctx)
		}
		return
	}
}

func FetchArchive(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	params := mux.Vars(r)
	t := model.Tag{}
	err := t.SetTag(params["tag"])
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
			http.Error(w, "Internal server error", 500)
			return
		}
		if expired {
			ctx.Log.Println("Expired: " + t.ExpirationReadable)
			http.Error(w, "This tag has expired.", 410)
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
			http.Error(w, "Error reading the tag contents.", 404)
			return
		}
	} else {
		// The tag does not exist
		http.Error(w, "Not found", 404)
		return
	}

	if cfg.TriggerDownloadTag != "" {
		ctx.Log.Println("Executing trigger: Download tag")
		triggerDownloadTagHandler(cfg.TriggerDownloadTag, t.Tag)
	}

	w.Header().Set("Cache-Control", "s-maxage=3600")

	var status = 200
	w.Header().Set("Content-Type", "application/zip")

	// Generate a map of paths to add to the zip response
	var paths []string
	for _, f := range t.Files {
		path := filepath.Join(f.TagDir, f.Filename)
		paths = append(paths, path)
	}
	output.ZIPresponse(w, status, t.Tag, paths, ctx)
	return
}

func ViewIndex(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	t := model.Tag{}
	tag := randomString(cfg.DefaultTagLength)
	err := t.SetTag(tag)
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}
	ctx.Log.Println("Tag generated: " + t.Tag)

	w.Header().Set("Cache-Control", "s-maxage=3600")
	w.Header().Set("Location", ctx.Baseurl+"/"+t.Tag)
	var status = 302
	output.JSONresponse(w, status, t, ctx)
}

//func Admin(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
//	http.Error(w, "Admin", 200)
//	return
//}

func PurgeHandler(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	ctx.Log.Println("Unexpected PURGE request received")
	http.Error(w, "Not implemented", 501)
	return
}

//func ViewAPI(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
//	t := model.Tag{}
//
//	w.Header().Set("Cache-Control", "s-maxage=3600")
//
//	var status = 200
//	output.HTMLresponse(w, "api", status, t, ctx)
//}

//func ViewDoc(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
//	t := model.Tag {}
//	headers := make(map[string]string)
//	headers["Cache-Control"] = "s-maxage=1"
//	var status = 200
//	output.HTMLresponse(w, "doc", status, headers, t, ctx)
//}
