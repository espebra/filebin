package api

import (
	"errors"
	"fmt"
	"github.com/espebra/filebin/app/backend/fs"
	"github.com/espebra/filebin/app/config"
	"github.com/espebra/filebin/app/metrics"
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
	"time"
	//"sort"
	"github.com/dustin/go-humanize"
)

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

func purgeURL(url string) error {
	timeout := time.Duration(2 * time.Second)
	client := &http.Client{
		Timeout: timeout,
	}
	fmt.Println("Purged " + url)

	// Invalidate the file
	req, err := http.NewRequest("PURGE", url, nil)
	if err != nil {
		return err
	}

	_, err = client.Do(req)
	if err != nil {
		return err
	}
	// Should probably log the URL and response code
	return nil
}

func Upload(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	bin := r.Header.Get("bin")
	if err := verifyBin(bin); err != nil {
		http.Error(w, "Invalid bin", 400)
		return
	}

	b, err := ctx.Backend.GetBinMetaData(bin)
	if err == nil {
		if b.Expired {
			http.Error(w, "This bin expired "+b.ExpiresReadable+".", 410)
			return
		}
	}

	filename := sanitizeFilename(r.Header.Get("filename"))
	if err := verifyFilename(filename); err != nil {
		http.Error(w, "Invalid filename", 400)
		return
	}

	ctx.Metrics.Incr("current-upload")
	defer ctx.Metrics.Decr("current-upload")

	event := metrics.Event {
	        Bin: bin,
	        Category: "file-upload",
	        Filename: filename,
	        RemoteAddr: ctx.RemoteAddr,
	        Text: r.Header.Get("user-agent"),
	        URL: r.RequestURI,
	}
	ctx.Metrics.AddEvent(event)

	f, err := ctx.Backend.UploadFile(bin, filename, r.Body)
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	ctx.Metrics.Incr("total-upload")

	if cfg.TriggerUploadFile != "" {
		ctx.Log.Println("Executing trigger: Uploaded file")
		triggerUploadFileHandler(cfg.TriggerUploadFile, f.Bin, f.Filename)
	}

	// Purging any old content
	if cfg.CacheInvalidation {
		for _, l := range f.Links {
			if err := purgeURL(l.Href); err != nil {
				ctx.Log.Println(err)
			}
		}
	}

	j := model.Job{}
	j.Filename = f.Filename
	j.Bin = f.Bin
	ctx.WorkQueue <- j

	w.Header().Set("Content-Type", "application/json")

	var status = 201
	output.JSONresponse(w, status, f, ctx)
}

func FetchFile(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	w.Header().Set("Cache-Control", "s-maxage=3600")
	w.Header().Set("Vary", "Content-Type")

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
		http.Error(w, "This bin expired "+b.ExpiresReadable+".", 410)
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

	ctx.Metrics.Incr("total-file-download")
	ctx.Metrics.Incr("current-file-download")
	defer ctx.Metrics.Decr("current-file-download")

	event := metrics.Event {
	        Bin: bin,
	        Category: "file-download",
	        Filename: filename,
	        RemoteAddr: ctx.RemoteAddr,
	        Text: r.Header.Get("user-agent"),
	        URL: r.RequestURI,
	}
	ctx.Metrics.AddEvent(event)

	fp, err := ctx.Backend.GetFile(bin, filename)
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w, "Not found", 404)
		return
	}

	w.Header().Set("Content-SHA256", f.Checksum)

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

	b, err := ctx.Backend.DeleteBin(bin)
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	ctx.Metrics.Incr("total-bin-delete")

	// Purging any old content
	if cfg.CacheInvalidation {
		for _, f := range b.Files {
			for _, l := range f.Links {
				if err := purgeURL(l.Href); err != nil {
					ctx.Log.Println(err)
				}
			}
		}
	}

	ctx.Log.Println("Bin deleted successfully.")
	w.Header().Set("Cache-Control", "s-maxage=0, max-age=0")
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

	f, err := ctx.Backend.DeleteFile(bin, filename)
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	ctx.Metrics.Incr("total-file-delete")

	if cfg.TriggerDeleteFile != "" {
		ctx.Log.Println("Executing trigger: Delete file")
		triggerDeleteFileHandler(cfg.TriggerDeleteFile, bin, filename)
	}

	// Purging any old content
	if cfg.CacheInvalidation {
		for _, l := range f.Links {
			if err := purgeURL(l.Href); err != nil {
				ctx.Log.Println(err)
			}
		}
	}

	w.Header().Set("Cache-Control", "s-maxage=0, max-age=0")
	http.Error(w, "File deleted successfully", 200)
	return
}

func FetchAlbum(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	w.Header().Set("Cache-Control", "s-maxage=3600")

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
		http.Error(w, "This bin expired "+b.ExpiresReadable+".", 410)
		return
	}

	ctx.Metrics.Incr("total-view-album")

	event := metrics.Event {
	        Bin: bin,
	        Category: "view-album",
	        RemoteAddr: ctx.RemoteAddr,
	        Text: r.Header.Get("user-agent"),
	        URL: r.RequestURI,
	}
	ctx.Metrics.AddEvent(event)

	var status = 200
	output.HTMLresponse(w, "viewalbum", status, b, ctx)
	return
}

func FetchBin(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	w.Header().Set("Cache-Control", "s-maxage=3600")
	w.Header().Set("Vary", "Content-Type")

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
		http.Error(w, "This bin expired "+b.ExpiresReadable+".", 410)
		return
	}

	ctx.Metrics.Incr("total-view-bin")

	event := metrics.Event {
	        Bin: bin,
	        Category: "view-bin",
	        RemoteAddr: ctx.RemoteAddr,
	        Text: r.Header.Get("user-agent"),
	        URL: r.RequestURI,
	}
	ctx.Metrics.AddEvent(event)

	var status = 200

	w.Header().Set("Cache-Control", "s-maxage=3600")
	w.Header().Set("Vary", "Content-Type")
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
	w.Header().Set("Cache-Control", "s-maxage=3600")
	w.Header().Set("Vary", "Content-Type")

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
		http.Error(w, "This bin expired "+b.ExpiresReadable+".", 410)
		return
	}

	ctx.Metrics.Incr("current-archive-download")
	defer ctx.Metrics.Decr("current-archive-download")

	event := metrics.Event {
	        Bin: bin,
	        Category: "archive-download",
	        RemoteAddr: ctx.RemoteAddr,
	        Text: r.Header.Get("user-agent"),
	        URL: r.RequestURI,
	}
	ctx.Metrics.AddEvent(event)

	_, _, err = ctx.Backend.GetBinArchive(bin, format, w)
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w, "Bin not found", 404)
		return
	}

	ctx.Metrics.Incr("total-archive-download")

	if cfg.TriggerDownloadBin != "" {
		ctx.Log.Println("Executing trigger: Download bin")
		triggerDownloadBinHandler(cfg.TriggerDownloadBin, bin)
	}
}

func ViewIndex(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	bin := randomString(cfg.DefaultBinLength)
	w.Header().Set("Location", ctx.Baseurl+"/"+bin)
	w.Header().Set("Cache-Control", "s-maxage=0, max-age=0")
	http.Error(w, "Generated bin: "+bin, 302)
	return
}

func AdminDashboard(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	w.Header().Set("Vary", "Content-Type")
	w.Header().Set("Cache-Control", "s-maxage=0, max-age=0")
	var status = 200

	event := metrics.Event {
	        Category: "admin-login",
	        RemoteAddr: ctx.RemoteAddr,
	        Text: r.Header.Get("user-agent"),
	        URL: r.RequestURI,
	}
	ctx.Metrics.AddEvent(event)

	bins := ctx.Backend.GetBinsMetaData()
	stats := ctx.Metrics.GetStats()

	filter := metrics.Event {
	        Category: "admin-login",
	}

	logins := ctx.Metrics.GetEvents(filter, time.Time{}, 3)

	limitTime := time.Now().UTC().Add(-48 * time.Hour)
	if len(logins) >= 2 {
		limitTime = logins[1].Timestamp
	}

	recentEvents := ctx.Metrics.GetEvents(metrics.Event{}, limitTime, 0)
	_, recentEvents = recentEvents[0], recentEvents[1:]

	filter = metrics.Event {
	        Category: "file-upload",
	}
	recentUploads := ctx.Metrics.GetEvents(filter, limitTime, 0)

	type Out struct {
		Bins           []fs.Bin
		BinsReadable   string
		Events         []metrics.Event
		Uploads        []metrics.Event
		Files          int
		FilesReadable  string
		Bytes          int64
		BytesReadable  string
		Stats          map[string]int64
		Logins         []metrics.Event
		Uptime         time.Duration
		UptimeReadable string
		Now            time.Time
	}

	var files int
	var bytes int64
	for _, b := range bins {
		files = files + len(b.Files)
		bytes = bytes + b.Bytes
	}

	data := Out{
		Bins:           bins,
		Events:         recentEvents,
		Uploads:        recentUploads,
		Files:          files,
		Bytes:          bytes,
		BytesReadable:  humanize.Bytes(uint64(bytes)),
		BinsReadable:   humanize.Comma(int64(len(bins))),
		FilesReadable:  humanize.Comma(int64(files)),
		Stats:          stats,
		Logins:         logins,
		Uptime:         ctx.Metrics.Uptime(),
		UptimeReadable: humanize.Time(ctx.Metrics.StartTime()),
		Now:            time.Now().UTC(),
	}

	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		output.JSONresponse(w, status, data, ctx)
	} else {
		output.HTMLresponse(w, "dashboard", status, data, ctx)
	}
	return
}

func AdminEvents(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	w.Header().Set("Vary", "Content-Type")
	w.Header().Set("Cache-Control", "s-maxage=0, max-age=0")
	var status = 200

	event := metrics.Event {
	        Category: "admin-login",
	        RemoteAddr: ctx.RemoteAddr,
	        Text: r.Header.Get("user-agent"),
	        URL: r.RequestURI,
	}
	ctx.Metrics.AddEvent(event)

	u, err := url.Parse(r.RequestURI)
	if err != nil {
		ctx.Log.Println(err)
	}

	queryParams, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		ctx.Log.Println(err)
	}

	filter := metrics.Event {
	        Bin: queryParams.Get("bin"),
	        Category: queryParams.Get("category"),
	        Filename: queryParams.Get("filename"),
	        RemoteAddr: queryParams.Get("remoteaddr"),
	        URL: queryParams.Get("url"),
	}

	events := ctx.Metrics.GetEvents(filter, time.Time{}, 100)

	type Out struct {
		Events         []metrics.Event
		Uptime         time.Duration
		UptimeReadable string
		Now            time.Time
	}

	data := Out{
		Events:         events,
		Uptime:         ctx.Metrics.Uptime(),
		UptimeReadable: humanize.Time(ctx.Metrics.StartTime()),
		Now:            time.Now().UTC(),
	}

	w.Header().Set("Cache-Control", "s-maxage=0, max-age=0")
	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		output.JSONresponse(w, status, data, ctx)
	} else {
		output.HTMLresponse(w, "events", status, data, ctx)
	}
	return
}

func AdminBins(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	w.Header().Set("Vary", "Content-Type")
	w.Header().Set("Cache-Control", "s-maxage=0, max-age=0")
	var status = 200

	event := metrics.Event {
	        Category: "admin-login",
	        RemoteAddr: ctx.RemoteAddr,
	        Text: r.Header.Get("user-agent"),
	        URL: r.RequestURI,
	}
	ctx.Metrics.AddEvent(event)

	bins := ctx.Backend.GetBinsMetaData()

	type Out struct {
		Bins           []fs.Bin
		BinsReadable   string
		Uptime         time.Duration
		UptimeReadable string
		Now            time.Time
	}

	data := Out{
		Bins:           bins,
		BinsReadable:   humanize.Comma(int64(len(bins))),
		Uptime:         ctx.Metrics.Uptime(),
		UptimeReadable: humanize.Time(ctx.Metrics.StartTime()),
		Now:            time.Now().UTC(),
	}

	w.Header().Set("Cache-Control", "s-maxage=0, max-age=0")
	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		output.JSONresponse(w, status, data, ctx)
	} else {
		output.HTMLresponse(w, "bins", status, data, ctx)
	}
	return
}

func PurgeHandler(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	ctx.Log.Println("Unexpected PURGE request received: " + r.RequestURI)
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

func FilebinStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "s-maxage=0, max-age=0")
	http.Error(w, "OK", 200)
	return
}
