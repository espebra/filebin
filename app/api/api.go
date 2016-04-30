package api

import (
	"fmt"
	"github.com/espebra/filebin/app/config"
	"github.com/espebra/filebin/app/model"
	"github.com/espebra/filebin/app/output"
	"github.com/gorilla/mux"
	"math/rand"
	"net/http"
	"strconv"
	"net/url"
	"os/exec"
	"regexp"
	"sort"
)

func isWorkaroundNeeded(useragent string) bool {
	matched, err := regexp.MatchString("(iPhone|iPad|iPod)", useragent)
	if err != nil {
		fmt.Println(err)
	}
	return matched
}

func triggerNewBinHandler(c string, bin string) error {
	cmd := exec.Command(c, bin)
	err := cmdHandler(cmd)
	return err
}

func triggerUploadFileHandler(c string, bin string, filename string) error {
	cmd := exec.Command(c, bin, filename)
	err := cmdHandler(cmd)
	return err
}

func triggerDownloadBinHandler(c string, bin string) error {
	cmd := exec.Command(c, bin)
	err := cmdHandler(cmd)
	return err
}

func triggerDownloadFileHandler(c string, bin string, filename string) error {
	cmd := exec.Command(c, bin, filename)
	err := cmdHandler(cmd)
	return err
}

func triggerDeleteBinHandler(c string, bin string) error {
	cmd := exec.Command(c, bin)
	err := cmdHandler(cmd)
	return err
}

func triggerDeleteFileHandler(c string, bin string, filename string) error {
	cmd := exec.Command(c, bin, filename)
	err := cmdHandler(cmd)
	return err
}

func triggerExpiredBinHandler(c string, bin string) error {
	cmd := exec.Command(c, bin)
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
	//params := mux.Vars(r)
	bin := r.Header.Get("bin")
	filename := r.Header.Get("filename")
	f, err := ctx.Backend.UploadFile(bin, filename, r.Body)
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	//f.RemoteAddr = r.RemoteAddr
	//f.UserAgent = r.Header.Get("User-Agent")

	// Extract the bin from the request
	//bin := r.Header.Get("bin")
	//if bin == "" {
	//	// To ensure backwards compatibility for uploads
	//	bin = r.Header.Get("tag")
	//}

	//if bin == "" {
	//	bin := randomString(cfg.DefaultBinLength)
	//	err = f.SetBin(bin)
	//	ctx.Log.Println("Bin generated: " + f.Bin)
	//} else {
	//	err = f.SetBin(bin)
	//	ctx.Log.Println("Bin specified: " + bin)
	//}
	//if err != nil {
	//	ctx.Log.Println(err)
	//	http.Error(w, err.Error(), http.StatusBadRequest)
	//	return
	//}
	//f.SetBinDir(cfg.Filedir)
	//ctx.Log.Println("Bin directory: " + f.BinDir)
	//contentLength, err := strconv.ParseUint(r.Header.Get("Content-Length"), 10, 64)
	//if err != nil {
	//	ctx.Log.Println(err)
	//}
	//ctx.Log.Printf("Specified content length: %d bytes", contentLength)

	// Read the amounts of bytes free in the filedir directory
	//var stat syscall.Statfs_t
	//syscall.Statfs(cfg.Filedir, &stat)
	//freeBytes := stat.Bavail * uint64(stat.Bsize)
	//ctx.Log.Printf("Free storage: %d bytes", freeBytes)

	//if contentLength >= freeBytes {
	//	ctx.Log.Println("Not enough free disk space for the specified content-length. Trying to abort here.")

	//	// This will not work as expected since clients (usually) don't care about
	//	// the response until the request delivery is complete.
	//	http.Error(w, "Request Entity Too Large", 413)
	//	return
	//}

	// Write the request body to a temporary file
	//err = f.WriteTempfile(r.Body, cfg.Tempdir)
	//if err != nil {
	//	ctx.Log.Println("Unable to write tempfile: ", err)

	//	// Clean up by removing the tempfile
	//	f.ClearTemp()

	//	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	//	return
	//}
	//ctx.Log.Println("Tempfile: " + f.Tempfile)
	//ctx.Log.Println("Tempfile size: " + strconv.FormatInt(f.Bytes, 10) + " bytes")

	//// Do not accept files that are 0 bytes
	//if f.Bytes == 0 {
	//	ctx.Log.Println("Empty files are not allowed. Aborting.")

	//	// Clean up by removing the tempfile
	//	f.ClearTemp()

	//	http.Error(w, "No content. The file size must be more than "+
	//		"0 bytes.", http.StatusBadRequest)
	//	return
	//}

	//// Calculate and verify the checksum
	//checksum := r.Header.Get("content-sha256")
	//if checksum != "" {
	//	ctx.Log.Println("Checksum specified: " + checksum)
	//}
	//err = f.VerifySHA256(checksum)
	//ctx.Log.Println("Checksum calculated: " + f.Checksum)
	//if err != nil {
	//	ctx.Log.Println("The specified checksum did not match")
	//	http.Error(w, "Checksum did not match", http.StatusConflict)
	//	return
	//}

	//// Trigger new bin
	//t := model.Bin{}
	//t.SetBin(f.Bin)
	//t.SetBinDir(cfg.Filedir)
	//if !t.BinDirExists() {
	//	if cfg.TriggerNewBin != "" {
	//		ctx.Log.Println("Executing trigger: New bin")
	//		triggerNewBinHandler(cfg.TriggerNewBin, f.Bin)
	//	}
	//}

	//// Create the bin directory if it does not exist
	//err = f.EnsureBinDirectoryExists()
	//if err != nil {
	//	ctx.Log.Println("Unable to create bin directory: ", f.BinDir)
	//	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	//	return
	//}

	//t.CalculateExpiration(cfg.Expiration)
	//expired, err := t.IsExpired(cfg.Expiration)
	//if err != nil {
	//	ctx.Log.Println(err)
	//	http.Error(w, "Internal server error", 500)
	//	return
	//}
	//if expired {
	//	ctx.Log.Println("The bin has expired. Aborting.")
	//	http.Error(w, "This bin has expired.", 410)
	//	return
	//}

	//// Extract the filename from the request
	//fname := r.Header.Get("filename")
	//if fname == "" {
	//	ctx.Log.Println("Filename generated: " + f.Checksum)
	//	f.SetFilename(f.Checksum)
	//} else {
	//	ctx.Log.Println("Filename specified: " + fname)
	//	err = f.SetFilename(fname)
	//	if err != nil {
	//		ctx.Log.Println(err)
	//		http.Error(w, "Invalid filename specified. It contains illegal characters or is too short.",
	//			http.StatusBadRequest)
	//		return
	//	}
	//}

	//if fname != f.Filename {
	//	ctx.Log.Println("Filename sanitized: " + f.Filename)
	//}

	//err = f.DetectMIME()
	//if err != nil {
	//	ctx.Log.Println("Unable to detect MIME: ", err)
	//} else {
	//	ctx.Log.Println("MIME detected: " + f.MIME)
	//}

	//ctx.Log.Println("Media type: " + f.MediaType())
	//if f.MediaType() == "image" {
	//	err = f.ParseExif()
	//	if err != nil {
	//		ctx.Log.Println(err)
	//	}

	//	// iOS devices provide only one filename even when uploading
	//	// multiple images. Providing some workaround for this below.
	//	// XXX: Refactoring needed.
	//	if isWorkaroundNeeded(f.UserAgent) && !f.DateTime.IsZero() {
	//		var fname string
	//		dt := f.DateTime.Format("060102-150405")

	//		// List of filenames to modify
	//		if f.Filename == "image.jpeg" {
	//			fname = "img-" + dt + ".jpeg"
	//		}
	//		if f.Filename == "image.gif" {
	//			fname = "img-" + dt + ".gif"
	//		}
	//		if f.Filename == "image.png" {
	//			fname = "img-" + dt + ".png"
	//		}

	//		if fname != "" {
	//			ctx.Log.Println("Filename workaround triggered")
	//			ctx.Log.Println("Filename modified: " + fname)
	//			err = f.SetFilename(fname)
	//			if err != nil {
	//				ctx.Log.Println(err)
	//			}
	//		}
	//	}

	//	//err = f.GenerateThumbnail()
	//	//if err != nil {
	//	//	ctx.Log.Println(err)
	//	//}

	//	//extra := make(map[string]string)
	//	//if !f.DateTime.IsZero() {
	//	//	extra["DateTime"] = f.DateTime.String()
	//	//}
	//	//f.Extra = extra
	//}

	//// Promote file from tempdir to the published bindir
	//f.Publish()

	//// Clean up by removing the tempfile
	//f.ClearTemp()

	//err = f.StatInfo()
	//if err != nil {
	//	ctx.Log.Println(err)
	//	http.Error(w, "Internal Server Error", 500)
	//	return
	//}

	//f.GenerateLinks(cfg.Baseurl)
	//f.CreatedAt = time.Now().UTC()
	////f.ExpiresAt = time.Now().UTC().Add(24 * 7 * 4 * time.Hour)

	if cfg.TriggerUploadFile != "" {
		ctx.Log.Println("Executing trigger: Uploaded file")
		triggerUploadFileHandler(cfg.TriggerUploadFile, bin, filename)
	}

	// Purging any old content
	//if cfg.CacheInvalidation {
	//	if err := f.Purge(); err != nil {
	//		ctx.Log.Println(err)
	//	}
	//}

	j := model.Job { }
	j.Filename = filename
	j.Bin = bin
	ctx.WorkQueue <- j

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

	params := mux.Vars(r)
	bin := params["bin"]
	filename := params["filename"]

	f, err := ctx.Backend.GetFileMetaData(bin, filename)
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w, "Not found", 404)
		return
	}

	w.Header().Set("Vary", "Content-Type")
	w.Header().Set("Cache-Control", "s-maxage=3600")
	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		output.JSONresponse(w, 200, f, ctx)
		return
	}

	width, _ := strconv.Atoi(queryParams.Get("width"))
	height, _ := strconv.Atoi(queryParams.Get("height"))
	if (width > 0) || (height > 0) {
	        fp, err := ctx.Backend.GetThumbnail(bin, filename, width, height)
		if err != nil {
			ctx.Log.Println(err)
			http.Error(w, "Image not found", 404)
			return
		}
		http.ServeContent(w, r, f.Filename, f.CreatedAt, fp)
		return
	}

	fp, err := ctx.Backend.GetFile(bin, filename)
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w, "Not found", 404)
		return
	}

	if f.Algorithm == "sha256" {
		w.Header().Set("Content-SHA256", f.Checksum)
	}

	if cfg.TriggerDownloadFile != "" {
		ctx.Log.Println("Executing trigger: Download file")
		triggerDownloadFileHandler(cfg.TriggerDownloadFile, bin, filename)
	}

	http.ServeContent(w, r, f.Filename, f.CreatedAt, fp)
}

func DeleteBin(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	params := mux.Vars(r)
	bin := params["bin"]

	if err := ctx.Backend.DeleteBin(bin); err != nil {
		ctx.Log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	//// Purging any old content
	//if cfg.CacheInvalidation {
	//	for _, f := range t.Files {
	//		if err := f.Purge(); err != nil {
	//			ctx.Log.Println(err)
	//		}
	//	}
	//}

	ctx.Log.Println("Bin deleted successfully.")
	http.Error(w, "Bin Deleted Successfully", 200)
	return

}

func DeleteFile(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	params := mux.Vars(r)
	bin := params["bin"]
	filename := params["filename"]

	if err := ctx.Backend.DeleteFile(bin, filename); err != nil {
		ctx.Log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	if cfg.TriggerDeleteFile != "" {
		ctx.Log.Println("Executing trigger: Delete file")
		triggerDeleteFileHandler(cfg.TriggerDeleteFile, bin, filename)
	}

	//// Purging any old content
	//if cfg.CacheInvalidation {
	//	if err := f.Purge(); err != nil {
	//		ctx.Log.Println(err)
	//	}
	//}

	http.Error(w, "File deleted successfully", 200)
}

func FetchAlbum(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	t := model.Bin{}
	params := mux.Vars(r)
	err := t.SetBin(params["bin"])
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w, "Invalid bin", 400)
		return
	}

	t.SetBinDir(cfg.Filedir)
	t.CalculateExpiration(cfg.Expiration)
	if t.BinDirExists() {
		expired, err := t.IsExpired(cfg.Expiration)
		if err != nil {
			ctx.Log.Println(err)
			http.Error(w, "Internal server error", 500)
			return
		}
		if expired {
			ctx.Log.Println("Expired: " + t.ExpirationReadable())
			http.Error(w, "This bin has expired.", 410)
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
			http.Error(w, "Error reading the bin contents.", 404)
			return
		}
	} else {
		// The bin does not exist
		http.Error(w, "Not found", 404)
		return
	}

	w.Header().Set("Cache-Control", "s-maxage=3600")

	var status = 200
	output.HTMLresponse(w, "viewalbum", status, t, ctx)
	return
}

func FetchBin(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	params := mux.Vars(r)
	bin := params["bin"]

	var err error

	b, err := ctx.Backend.GetBinMetaData(bin)
	if err != nil {
		if ctx.Backend.BinExists(bin) {
			ctx.Log.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		} else {
			// This bin needs to be created
			b = ctx.Backend.NewBin(bin)
		}
	}

	//t.SetBinDir(cfg.Filedir)
	//t.CalculateExpiration(cfg.Expiration)
	//if t.BinDirExists() {
	//	expired, err := t.IsExpired(cfg.Expiration)
	//	if err != nil {
	//		ctx.Log.Println(err)
	//		http.Error(w, "Internal server error", 500)
	//		return
	//	}
	//	if expired {
	//		ctx.Log.Println("Expired: " + t.ExpirationReadable())
	//		http.Error(w, "This bin has expired.", 410)
	//		return
	//	}

	//	err = t.StatInfo()
	//	if err != nil {
	//		ctx.Log.Println(err)
	//		http.Error(w, "Internal Server Error", 500)
	//		return
	//	}

	//	err = t.List(cfg.Baseurl)
	//	if err != nil {
	//		ctx.Log.Println(err)
	//		http.Error(w, "Error reading the bin contents.", 404)
	//		return
	//	}
	//}

	w.Header().Set("Vary", "Content-Type")
	w.Header().Set("Cache-Control", "s-maxage=3600")

	var status = 200

	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		output.JSONresponse(w, status, b, ctx)
		return
	} else {
		if len(b.Files) == 0 {
			output.HTMLresponse(w, "newbin", status, b, ctx)
		} else {
			output.HTMLresponse(w, "viewbin", status, b, ctx)
		}
		return
	}
}

func FetchArchive(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	params := mux.Vars(r)
	format := params["format"]
	bin := params["bin"]

	_, _, err := ctx.Backend.GetBinArchive(bin, format, w)
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w, "Bin not found", 404)
		return
	}

	if cfg.TriggerDownloadBin != "" {
		ctx.Log.Println("Executing trigger: Download bin")
		triggerDownloadBinHandler(cfg.TriggerDownloadBin, bin)
	}

	w.Header().Set("Cache-Control", "s-maxage=3600")

	// Generate a map of paths to add to the tar response
	//var paths []string
	//for _, f := range t.Files {
	//	path := filepath.Join(f.BinDir, f.Filename)
	//	paths = append(paths, path)
	//}

	//if format == "zip" {
	//	output.ZIPresponse(w, t.Bin, paths, ctx)
	//} else if format == "tar" {
	//	output.TARresponse(w, t.Bin, paths, ctx)
	//} else {
	//	http.Error(w, "Not found", 404)
	//}
	//http.ServeContent(w, r, archiveName, time.Now(), fp)
	//bytes, err := io.Copy(w, fp)
	//if err != nil {
	//	ctx.Log.Println(err)
	//	http.Error(w, "Archive generation failed", 500)
	//	return
	//}
}

func ViewIndex(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	t := model.Bin{}
	bin := randomString(cfg.DefaultBinLength)
	err := t.SetBin(bin)
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}
	ctx.Log.Println("Bin generated: " + t.Bin)

	w.Header().Set("Cache-Control", "s-maxage=3600")
	w.Header().Set("Location", ctx.Baseurl+"/"+t.Bin)
	var status = 302
	output.JSONresponse(w, status, t, ctx)
}

func Admin(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	bins := model.Bins{}
	bins.Baseurl = cfg.Baseurl
	bins.Expiration = cfg.Expiration
	bins.Filedir = cfg.Filedir
	if err := bins.Scan(); err != nil {
		ctx.Log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	sort.Sort(model.BinsByLastUpdate(bins.Bins))

	var status = 200
	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		output.JSONresponse(w, status, bins, ctx)
	} else {
		output.HTMLresponse(w, "admin", status, bins, ctx)
	}
	return
}

func PurgeHandler(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	ctx.Log.Println("Unexpected PURGE request received")
	http.Error(w, "Not implemented", 501)
	return
}

//func ViewAPI(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
//	t := model.Bin{}
//
//	w.Header().Set("Cache-Control", "s-maxage=3600")
//
//	var status = 200
//	output.HTMLresponse(w, "api", status, t, ctx)
//}

//func ViewDoc(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
//	t := model.Bin {}
//	headers := make(map[string]string)
//	headers["Cache-Control"] = "s-maxage=1"
//	var status = 200
//	output.HTMLresponse(w, "doc", status, headers, t, ctx)
//}
