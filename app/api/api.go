package api

import (
	"errors"
	"fmt"
	"github.com/espebra/filebin/app/backend/fs"
	"github.com/espebra/filebin/app/config"
	"github.com/espebra/filebin/app/model"
	"github.com/espebra/filebin/app/output"
	"github.com/gorilla/mux"
	"math/rand"
	"net/http"
	"net/url"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	//"sort"
	"github.com/dustin/go-humanize"
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

func verifyBin(s string) error {
	var invalid = regexp.MustCompile("[^A-Za-z0-9-_.]")
	if invalid.MatchString(s) {
		return errors.New("The bin contains invalid characters.")
	}

	if len(s) < 8 {
		return errors.New("The bin is too short.")
	}

	if strings.HasPrefix(s, ".") {
		return errors.New("Invalid bin specified.")
	}

	return nil
}

func verifyFilename(s string) error {
	var invalid = regexp.MustCompile("[^A-Za-z0-9-_=,.]")
	if invalid.MatchString(s) {
		return errors.New("The filename contains invalid characters.")
	}

	if len(s) == 0 {
		return errors.New("The filename is empty.")
	}

	if strings.HasPrefix(s, ".") {
		return errors.New("Invalid filename specified.")
	}

	return nil
}

func sanitizeFilename(s string) string {
	var invalid = regexp.MustCompile("[^A-Za-z0-9-_=,.]")
	s = invalid.ReplaceAllString(s, "_")

	if strings.HasPrefix(s, ".") {
		s = strings.Replace(s, ".", "_", 1)
	}

	if len(s) == 0 {
		s = "_"
	}
	return s
}

func Upload(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	//params := mux.Vars(r)
	bin := r.Header.Get("bin")
	if err := verifyBin(bin); err != nil {
		http.Error(w, "Invalid bin", 400)
		return
	}

	b, err := ctx.Backend.GetBinMetaData(bin)
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w, "Not found", 404)
		return
	}

	if b.Expired {
		http.Error(w, "This bin expired " + b.ExpiresReadable + ".", 410)
		return
	}

	filename := sanitizeFilename(r.Header.Get("filename"))
	if err := verifyFilename(filename); err != nil {
		http.Error(w, "Invalid filename", 400)
		return
	}

	f, err := ctx.Backend.UploadFile(bin, filename, r.Body)
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	//f.RemoteAddr = r.RemoteAddr
	//f.UserAgent = r.Header.Get("User-Agent")

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

	j := model.Job{}
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
	if err := verifyBin(bin); err != nil {
		http.Error(w, "Invalid bin", 400)
		return
	}

	b, err := ctx.Backend.GetBinMetaData(bin)
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w, "Not found", 404)
		return
	}

	if b.Expired {
		http.Error(w, "This bin expired " + b.ExpiresReadable + ".", 410)
		return
	}

	filename := params["filename"]
	if err := verifyFilename(filename); err != nil {
		http.Error(w, "Invalid filename", 400)
		return
	}

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
	if err := verifyBin(bin); err != nil {
		http.Error(w, "Invalid bin", 400)
		return
	}

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
	if err := verifyBin(bin); err != nil {
		http.Error(w, "Invalid bin", 400)
		return
	}

	filename := params["filename"]
	if err := verifyFilename(filename); err != nil {
		http.Error(w, "Invalid filename", 400)
		return
	}

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
	params := mux.Vars(r)
	bin := params["bin"]
	if err := verifyBin(bin); err != nil {
		http.Error(w, "Invalid bin", 400)
		return
	}

	b, err := ctx.Backend.GetBinMetaData(bin)
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if b.Expired {
		http.Error(w, "This bin expired " + b.ExpiresReadable + ".", 410)
		return
	}

	w.Header().Set("Cache-Control", "s-maxage=3600")

	var status = 200
	output.HTMLresponse(w, "viewalbum", status, b, ctx)
	return
}

func FetchBin(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	params := mux.Vars(r)
	bin := params["bin"]
	if err := verifyBin(bin); err != nil {
		http.Error(w, "Invalid bin", 400)
		return
	}

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

	if b.Expired {
		http.Error(w, "This bin expired " + b.ExpiresReadable + ".", 410)
		return
	}

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
	if err := verifyBin(bin); err != nil {
		http.Error(w, "Invalid bin", 400)
		return
	}

	b, err := ctx.Backend.GetBinMetaData(bin)
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w, "Not found", 404)
		return
	}

	if b.Expired {
		http.Error(w, "This bin expired " + b.ExpiresReadable + ".", 410)
		return
	}

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
}

func ViewIndex(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	bin := randomString(cfg.DefaultBinLength)
	w.Header().Set("Location", ctx.Baseurl+"/"+bin)
	http.Error(w, "Generated bin: "+bin, 302)
	return
}

func Admin(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	var status = 200
	bins := ctx.Backend.GetBinsMetaData()

	type Out struct {
		Bins          []fs.Bin
		Files         int
		Bytes         int64
		BytesReadable string
	}

	var files int
	var bytes int64
	for _, b := range bins {
		files = files + len(b.Files)
		bytes = bytes + b.Bytes
	}

	data := Out{
		Bins:          bins,
		Files:         files,
		Bytes:         bytes,
		BytesReadable: humanize.Bytes(uint64(bytes)),
	}

	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		output.JSONresponse(w, status, data, ctx)
	} else {
		output.HTMLresponse(w, "admin", status, data, ctx)
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
