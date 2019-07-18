package api

import (
	"errors"
	"github.com/dustin/go-humanize"
	"github.com/espebra/filebin/app/backend/fs"
	"github.com/espebra/filebin/app/config"
	"github.com/espebra/filebin/app/events"
	"github.com/espebra/filebin/app/model"
	"github.com/espebra/filebin/app/output"
	"github.com/espebra/filebin/app/shared"
	"github.com/espebra/filebin/app/tokens"
	"github.com/gorilla/mux"
	"math/rand"
	"net/http"
	"net/url"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
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
	err := cmd.Run()
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
	r.Close = true

	bin := r.Header.Get("bin")
	if bin == "" {
		// XXX: Should ensure that the bin does not exist from before.
		bin = randomString(cfg.DefaultBinLength)
	}
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

	event := ctx.Events.New(ctx.RemoteAddr, []string{"file", "upload"}, bin, filename)
	defer event.Done()

	if i, err := strconv.Atoi(r.Header.Get("content-length")); err == nil {
		event.Update("Size: "+humanize.Bytes(uint64(i)), 0)
	}

	if ctx.Backend.BinExists(bin) == false {
		if cfg.TriggerNewBin != "" {
			ctx.Log.Println("Executing trigger: New bin")
			triggerNewBinHandler(cfg.TriggerNewBin, bin)
		}
	}

	f, err := ctx.Backend.UploadFile(bin, filename, r.Body)
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		event.Update(err.Error(), 2)
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
			if err := shared.PurgeURL(l.Href, ctx.Log); err != nil {
				ctx.Log.Println(err)
			}
		}
	}

	j := model.Job{}
	j.Filename = f.Filename
	j.Bin = f.Bin
	j.Log = ctx.Log
	j.Cfg = &cfg
	ctx.WorkQueue <- j

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Filename", f.Filename)
	w.Header().Set("Bin", f.Bin)

	var status = 201
	output.JSONresponse(w, status, f, ctx)
}

func FetchFile(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	w.Header().Set("Cache-Control", "s-maxage=1")
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

	w.Header().Set("Bin", bin)
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

	w.Header().Set("Filename", filename)
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
		ctx.Metrics.Incr("total-thumbnails-viewed")
		http.ServeContent(w, r, f.Filename, f.CreatedAt, fp)
		return
	}

	if cfg.HotLinking == false {
		token := r.URL.Query().Get("t")
		if token == "" {
			token = r.Header.Get("token")
		}
		if ctx.Tokens.Verify(token) == false {
			ctx.Token = ctx.Tokens.Generate()
			f.Links = UpdateLinks(f.Links, ctx.Token)

			status := 200
			output.HTMLresponse(w, "invalidtokenfile", status, f, ctx)
			return
		}
	}

	event := ctx.Events.New(ctx.RemoteAddr, []string{"file", "download"}, bin, filename)
	defer event.Done()
	event.Update(humanize.Bytes(uint64(f.Bytes)), 0)

	ctx.Metrics.Incr("total-file-download")
	ctx.Metrics.Incr("current-file-download")
	defer ctx.Metrics.Decr("current-file-download")
	ctx.Metrics.Incr("file-download bin=" + bin + " filename=" + filename)

	fp, err := ctx.Backend.GetFile(bin, filename)
	if err != nil {
		ctx.Log.Println(err)
		event.Update(err.Error(), 2)
		http.Error(w, "Not found", 404)
		return
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

	b, err := ctx.Backend.DeleteBin(bin)
	if err != nil {
		ctx.Log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	ctx.Metrics.Incr("total-bin-delete")

	event := ctx.Events.New(ctx.RemoteAddr, []string{"bin", "delete"}, bin, "")
	defer event.Done()

	// Purging any old content
	if cfg.CacheInvalidation {
		for _, f := range b.Files {
			for _, l := range f.Links {
				if err := shared.PurgeURL(l.Href, ctx.Log); err != nil {
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

	event := ctx.Events.New(ctx.RemoteAddr, []string{"file", "delete"}, bin, filename)
	defer event.Done()

	if cfg.TriggerDeleteFile != "" {
		ctx.Log.Println("Executing trigger: Delete file")
		triggerDeleteFileHandler(cfg.TriggerDeleteFile, bin, filename)
	}

	// Purging any old content
	if cfg.CacheInvalidation {
		for _, l := range f.Links {
			if err := shared.PurgeURL(l.Href, ctx.Log); err != nil {
				ctx.Log.Println(err)
			}
		}
	}

	w.Header().Set("Cache-Control", "s-maxage=0, max-age=0")
	http.Error(w, "File deleted successfully", 200)
	return
}

func FetchAlbum(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	w.Header().Set("Cache-Control", "s-maxage=1")

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

	if cfg.HotLinking == false {
		ctx.Token = ctx.Tokens.Generate()
		for i, f := range b.Files {
			b.Files[i].Links = UpdateLinks(f.Links, ctx.Token)
		}
	}

	ctx.Metrics.Incr("total-view-album")
	ctx.Metrics.Incr("album-view bin=" + bin)

	var status = 200
	output.HTMLresponse(w, "viewalbum", status, b, ctx)
	return
}

func UpdateLinks(links []fs.Link, token string) []fs.Link {
	for i, l := range links {
		if l.Rel == "file" {
			u, err := url.Parse(l.Href)
			if err != nil {
				panic(err)
			}
			q := u.Query()
			q.Set("t", token)
			u.RawQuery = q.Encode()
			links[i].Href = u.String()
		}
	}
	return links
}

func FetchBin(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	w.Header().Set("Cache-Control", "s-maxage=1")
	w.Header().Set("Vary", "Content-Type")

	var status = 200

	params := mux.Vars(r)
	bin := params["bin"]
	if err := verifyBin(bin); err != nil {
		http.Error(w, "Invalid bin", 400)
		return
	}

	event := ctx.Events.New(ctx.RemoteAddr, []string{"bin", "view"}, bin, "")
	defer event.Done()

	b, err := ctx.Backend.GetBinMetaData(bin)
	if err != nil {
		if ctx.Backend.BinExists(bin) {
			ctx.Log.Println(err)
			event.Update(err.Error(), 2)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		} else {
			// This bin does not exist (but can be created)
			event.Update("Bin does not exist", 1)
			status = 404
			b = ctx.Backend.NewBin(bin)
		}
	}

	if b.Expired {
		http.Error(w, "This bin expired "+b.ExpiresReadable+".", 410)
		return
	}

	if cfg.HotLinking == false {
		ctx.Token = ctx.Tokens.Generate()
		for i, f := range b.Files {
			b.Files[i].Links = UpdateLinks(f.Links, ctx.Token)
		}
	}

	ctx.Metrics.Incr("total-view-bin")
	ctx.Metrics.Incr("bin-view bin=" + bin)

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
	w.Header().Set("Cache-Control", "s-maxage=1")
	w.Header().Set("Vary", "Content-Type")

	params := mux.Vars(r)
	format := params["format"]
	bin := params["bin"]
	if err := verifyBin(bin); err != nil {
		http.Error(w, "Invalid bin", 400)
		return
	}

	event := ctx.Events.New(ctx.RemoteAddr, []string{"archive", "download"}, bin, "")
	event.Update("Format: "+format, 0)
	defer event.Done()

	b, err := ctx.Backend.GetBinMetaData(bin)
	if err != nil {
		ctx.Log.Println(err)
		event.Update(err.Error(), 2)
		http.Error(w, "Not found", 404)
		return
	}

	if b.Expired {
		http.Error(w, "This bin expired "+b.ExpiresReadable+".", 410)
		event.Update("This bin expired"+b.ExpiresReadable, 2)
		return
	}

	if cfg.HotLinking == false {
		token := r.URL.Query().Get("t")
		if token == "" {
			token = r.Header.Get("token")
		}
		if ctx.Tokens.Verify(token) == false {
			// Token not set or invalid
			status := 200
			ctx.Token = ctx.Tokens.Generate()
			output.HTMLresponse(w, "invalidtokenarchive", status, b, ctx)
			return
		}
	}

	ctx.Metrics.Incr("current-archive-download")
	defer ctx.Metrics.Decr("current-archive-download")

	_, _, err = ctx.Backend.GetBinArchive(bin, format, w)
	if err != nil {
		ctx.Log.Println(err)
		event.Update(err.Error(), 2)
		http.Error(w, "Bin not found", 404)
		return
	}

	ctx.Metrics.Incr("total-archive-download")
	ctx.Metrics.Incr("archive-download bin=" + bin + " format=" + format)

	if cfg.TriggerDownloadBin != "" {
		ctx.Log.Println("Executing trigger: Download bin")
		triggerDownloadBinHandler(cfg.TriggerDownloadBin, bin)
	}
}

func NewBin(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	w.Header().Set("Cache-Control", "s-maxage=0, max-age=0")
	w.Header().Set("Vary", "Content-Type")

	// XXX: Should ensure that the bin does not exist from before.
	bin := randomString(cfg.DefaultBinLength)
	b := ctx.Backend.NewBin(bin)

	ctx.Metrics.Incr("total-new-bin")

	var status = 200

	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		output.JSONresponse(w, status, b, ctx)
	} else {
		output.HTMLresponse(w, "newbin", status, b, ctx)
	}
	return
}

func AdminDashboard(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	w.Header().Set("Vary", "Content-Type")
	w.Header().Set("Cache-Control", "s-maxage=0, max-age=0")
	var status = 200

	eventsInProgress := ctx.Events.GetEventsInProgress(0, 0)

	event := ctx.Events.New(ctx.RemoteAddr, []string{"admin", "dashboard"}, "", "")
	event.Update(r.Header.Get("user-agent"), 0)
	defer event.Done()

	logins := ctx.Events.GetEventsByTags([]string{"admin"}, 0, 3)

	bins := ctx.Backend.GetBinsMetaData()
	stats := ctx.Metrics.GetStats()

	// Detect time limit for showing recent events
	limitTime := time.Now().UTC().Add(-48 * time.Hour)
	if len(logins) >= 2 {
		limitTime = logins[1].StartTime()
	}

	var recentUploads []events.Event
	uploads := ctx.Events.GetEventsByTags([]string{"upload"}, 0, 0)
	for _, f := range uploads {
		if f.StartTime().After(limitTime) {
			if f.IsDone() && f.Status() == 0 {
				recentUploads = append(recentUploads, f)
			}
		}
	}

	var recentEvents []events.Event
	allEvents := ctx.Events.GetAllEvents(1, 0)
	for _, e := range allEvents {
		if e.StartTime().After(limitTime) {
			recentEvents = append(recentEvents, e)
		}
	}

	type Out struct {
		Bins             []fs.Bin
		BinsReadable     string
		Events           []events.Event
		EventsInProgress []events.Event
		Uploads          []events.Event
		Files            int
		FilesReadable    string
		Bytes            int64
		BytesReadable    string
		Stats            map[string]int64
		Logins           []events.Event
		Uptime           time.Duration
		Tokens           int
		UptimeReadable   string
		Now              time.Time
	}

	var files int
	var bytes int64
	for _, b := range bins {
		files = files + len(b.Files)
		bytes = bytes + b.Bytes
	}

	data := Out{
		Bins:             bins,
		Events:           recentEvents,
		EventsInProgress: eventsInProgress,
		Uploads:          recentUploads,
		Files:            files,
		Bytes:            bytes,
		BytesReadable:    humanize.Bytes(uint64(bytes)),
		BinsReadable:     humanize.Comma(int64(len(bins))),
		FilesReadable:    humanize.Comma(int64(files)),
		Stats:            stats,
		Logins:           logins,
		Uptime:           ctx.Metrics.Uptime(),
		UptimeReadable:   humanize.Time(ctx.Metrics.StartTime()),
		Tokens:           len(ctx.Tokens.GetAllTokens()),
		Now:              time.Now().UTC(),
	}

	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		output.JSONresponse(w, status, data, ctx)
	} else {
		output.HTMLresponse(w, "dashboard", status, data, ctx)
	}
	return
}

func AdminCounters(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	w.Header().Set("Vary", "Content-Type")
	w.Header().Set("Cache-Control", "s-maxage=0, max-age=0")
	var status = 200

	event := ctx.Events.New(ctx.RemoteAddr, []string{"admin", "counters"}, "", "")
	event.Update(r.Header.Get("user-agent"), 0)
	defer event.Done()

	stats := ctx.Metrics.GetStats()

	type Out struct {
		Counters       map[string]int64
		Uptime         time.Duration
		UptimeReadable string
		Now            time.Time
	}

	data := Out{
		Counters:       stats,
		Uptime:         ctx.Metrics.Uptime(),
		UptimeReadable: humanize.Time(ctx.Metrics.StartTime()),
		Now:            time.Now().UTC(),
	}

	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		output.JSONresponse(w, status, data, ctx)
	} else {
		output.HTMLresponse(w, "counters", status, data, ctx)
	}
	return
}

func AdminEvents(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	w.Header().Set("Vary", "Content-Type")
	w.Header().Set("Cache-Control", "s-maxage=0, max-age=0")
	var status = 200

	event := ctx.Events.New(ctx.RemoteAddr, []string{"admin", "events"}, "", "")
	event.Update(r.Header.Get("user-agent"), 0)
	defer event.Done()

	type Out struct {
		Events         []events.Event
		Uptime         time.Duration
		UptimeReadable string
		Now            time.Time
	}

	data := Out{
		Events:         ctx.Events.GetAllEvents(0, 10000),
		Uptime:         ctx.Metrics.Uptime(),
		UptimeReadable: humanize.Time(ctx.Metrics.StartTime()),
		Now:            time.Now().UTC(),
	}

	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		output.JSONresponse(w, status, data, ctx)
	} else {
		output.HTMLresponse(w, "events", status, data, ctx)
	}
	return
}

func AdminTokens(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	w.Header().Set("Vary", "Content-Type")
	w.Header().Set("Cache-Control", "s-maxage=0, max-age=0")
	var status = 200

	event := ctx.Events.New(ctx.RemoteAddr, []string{"admin", "tokens"}, "", "")
	event.Update(r.Header.Get("user-agent"), 0)
	defer event.Done()

	type Out struct {
		Tokens         []tokens.Token
		Uptime         time.Duration
		UptimeReadable string
		Now            time.Time
	}

	data := Out{
		Tokens:         ctx.Tokens.GetAllTokens(),
		Uptime:         ctx.Metrics.Uptime(),
		UptimeReadable: humanize.Time(ctx.Metrics.StartTime()),
		Now:            time.Now().UTC(),
	}

	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		output.JSONresponse(w, status, data, ctx)
	} else {
		output.HTMLresponse(w, "tokens", status, data, ctx)
	}
	return
}

func AdminBins(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	w.Header().Set("Vary", "Content-Type")
	w.Header().Set("Cache-Control", "s-maxage=0, max-age=0")
	var status = 200

	event := ctx.Events.New(ctx.RemoteAddr, []string{"admin", "bins"}, "", "")
	event.Update(r.Header.Get("user-agent"), 0)
	defer event.Done()

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

func Readme(w http.ResponseWriter, r *http.Request, cfg config.Configuration, ctx model.Context) {
	var status = 200
	w.Header().Set("Cache-Control", "s-maxage=3600")

	type Out struct {
		Uptime         time.Duration
		UptimeReadable string
		Now            time.Time
	}

	data := Out{
		Uptime:         ctx.Metrics.Uptime(),
		UptimeReadable: humanize.Time(ctx.Metrics.StartTime()),
		Now:            time.Now().UTC(),
	}
	output.HTMLresponse(w, "readme", status, data, ctx)
}

func FilebinStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "s-maxage=0, max-age=0")
	http.Error(w, "OK", 200)
	return
}
