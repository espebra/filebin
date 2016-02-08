package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/GeertJohan/go.rice"
	"github.com/gorilla/mux"

	"github.com/espebra/filebin/app/api"
	"github.com/espebra/filebin/app/config"
	"github.com/espebra/filebin/app/model"
)

var cfg = config.Global
var githash = "No githash provided"
var buildstamp = "No buildstamp provided"

var staticBox *rice.Box
var templateBox *rice.Box

// Initiate buffered channel for batch processing
var WorkQueue = make(chan model.File, 1000)

func isDir(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	if fi.IsDir() {
		return true
	} else {
		return false
	}
}

func generateReqId(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyz0123456789")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())

	flag.StringVar(&cfg.Baseurl, "baseurl",
		cfg.Baseurl, "Baseurl used when generating links.")

	flag.StringVar(&cfg.Filedir, "filedir",
		cfg.Filedir, "Directory to store uploaded files.")

	flag.StringVar(&cfg.Tempdir, "tempdir",
		cfg.Tempdir, "Directory to temporarily store files during upload.")

	//flag.StringVar(&cfg.Logdir, "logdir",
	//	cfg.Logdir, "Directory to write log files to.")

	//flag.StringVar(&cfg.Thumbdir, "thumbdir",
	//	cfg.Thumbdir, "Path to thumbnail directory")

	//flag.StringVar(&cfg.Database, "database",
	//	cfg.Database, "Path to database file")

	flag.StringVar(&cfg.Host, "host",
		cfg.Host, "Listen host.")

	flag.IntVar(&cfg.Port, "port",
		cfg.Port, "Listen port.")

	flag.IntVar(&cfg.Readtimeout, "readtimeout",
		cfg.Readtimeout, "Request read timeout in seconds.")

	flag.IntVar(&cfg.Writetimeout, "writetimeout",
		cfg.Writetimeout, "Response write timeout in seconds.")

	flag.IntVar(&cfg.Maxheaderbytes, "maxheaderbytes",
		cfg.Maxheaderbytes, "Max header size in bytes.")

	flag.BoolVar(&cfg.CacheInvalidation, "cache-invalidation",
		cfg.CacheInvalidation,
		"HTTP PURGE requests will be sent on every change if enabled.")

	flag.IntVar(&cfg.Workers, "workers",
		cfg.Workers, "Number of workers for background processing.")

	//flag.IntVar(&cfg.Pagination, "pagination",
	//	cfg.Pagination,
	//	"Files to show per page for pagination.")

	//flag.StringVar(&cfg.GeoIP2, "geoip2",
	//	cfg.GeoIP2, "Path to the GeoIP2 database file.")

	flag.Int64Var(&cfg.Expiration, "expiration",
		cfg.Expiration, "Tag expiration time in seconds after the last modification.")

	//flag.BoolVar(&cfg.Verbose, "verbose",
	//	cfg.Verbose, "Verbose output.")

	flag.StringVar(&cfg.TriggerNewTag, "trigger-new-tag",
		cfg.TriggerNewTag,
		"Command to execute when a tag is created.")

	flag.StringVar(&cfg.TriggerUploadFile,
		"trigger-upload-file",
		cfg.TriggerUploadFile,
		"Command to execute when a file is uploaded.")

	flag.StringVar(&cfg.TriggerDownloadTag,
		"trigger-download-tag",
		cfg.TriggerDownloadTag,
		"Command to execute when a tag archive is downloaded.")

	flag.StringVar(&cfg.TriggerDownloadFile,
		"trigger-download-file",
		cfg.TriggerDownloadFile,
		"Command to execute when a file is downloaded.")

	flag.StringVar(&cfg.TriggerDeleteTag,
		"trigger-delete-tag",
		cfg.TriggerDeleteTag,
		"Command to execute when a tag is deleted.")

	flag.StringVar(&cfg.TriggerDeleteFile,
		"trigger-delete-file",
		cfg.TriggerDeleteFile,
		"Command to execute when a file is deleted.")

	//	flag.StringVar(&cfg.TriggerExpiredTag, "trigger-expired-tag",
	//		cfg.TriggerExpiredTag,
	//		"Trigger to execute when a tag expires.")

	flag.BoolVar(&cfg.Version, "version",
		cfg.Version, "Show the version information.")

	flag.Parse()

	if cfg.Version {
		fmt.Println("Git Commit Hash: " + githash)
		fmt.Println("UTC Build Time: " + buildstamp)
		os.Exit(0)
	}

	//if (!IsDir(cfg.Logdir)) {
	//    fmt.Println("The specified log directory is not a directory: ",
	//        cfg.Logdir)
	//    os.Exit(2)
	//}

	if cfg.Port < 1 || cfg.Port > 65535 {
		log.Fatalln("Invalid port number, aborting.")
	}

	if cfg.Readtimeout < 1 || cfg.Readtimeout > 86400 {
		log.Fatalln("Invalid read timeout, aborting.")
	}

	if cfg.Writetimeout < 1 || cfg.Writetimeout > 86400 {
		log.Fatalln("Invalid write timeout, aborting.")
	}

	if cfg.Maxheaderbytes < 1 ||
		cfg.Maxheaderbytes > 2<<40 {
		log.Fatalln("Invalid max header bytes, aborting.")
	}

	if !isDir(cfg.Tempdir) {
		log.Fatalln("The directory " + cfg.Tempdir + " does not exist.")
	}

	if !isDir(cfg.Filedir) {
		log.Fatalln("The directory " + cfg.Filedir + " does not exist.")
	}

	//if _, err := os.Stat(cfg.GeoIP2); err == nil {
	//    gi, err = geoip2.Open(cfg.GeoIP2)
	//    if err != nil {
	//        Info.Print("Could not open GeoIP2 database ", cfg.GeoIP2,
	//            ": ", err)
	//    }
	//    defer gi.Close()
	//} else {
	//    Info.Print("GeoIP2 database does not exist: ", cfg.GeoIP2)
	//}
}

func main() {
	log := log.New(os.Stdout, "- ", log.LstdFlags)

	// Initialize boxes
	staticBox = rice.MustFindBox("static")
	templateBox = rice.MustFindBox("templates")

	log.Println("Listen host: " + cfg.Host)
	log.Println("Listen port: " + strconv.Itoa(cfg.Port))
	log.Println("Read timeout: " +
		strconv.Itoa(cfg.Readtimeout) + " seconds")
	log.Println("Write timeout: " +
		strconv.Itoa(cfg.Writetimeout) + " seconds")
	log.Println("Max header size: " +
		strconv.Itoa(cfg.Maxheaderbytes) + " bytes")
	log.Println("Cache invalidation enabled: " +
		strconv.FormatBool(cfg.CacheInvalidation))
	log.Println("Workers: " +
		strconv.Itoa(cfg.Workers))
	log.Println("Expiration time: " +
		strconv.FormatInt(cfg.Expiration, 10) + " seconds")
	log.Println("Files directory: " + cfg.Filedir)
	log.Println("Temp directory: " + cfg.Tempdir)
	//log.Println("Log directory: " + cfg.Logdir)
	log.Println("Baseurl: " + cfg.Baseurl)

	var trigger = cfg.TriggerNewTag
	if trigger == "" {
		trigger = "Not set"
	}
	log.Println("Trigger - New tag: " + trigger)

	trigger = cfg.TriggerUploadFile
	if trigger == "" {
		trigger = "Not set"
	}
	log.Println("Trigger - Upload file: " + trigger)

	trigger = cfg.TriggerDownloadTag
	if trigger == "" {
		trigger = "Not set"
	}
	log.Println("Trigger - Download tag: " + trigger)

	trigger = cfg.TriggerDownloadFile
	if trigger == "" {
		trigger = "Not set"
	}
	log.Println("Trigger - Download file: " + trigger)

	trigger = cfg.TriggerDeleteTag
	if trigger == "" {
		trigger = "Not set"
	}
	log.Println("Trigger - Delete tag: " + trigger)

	trigger = cfg.TriggerDeleteFile
	if trigger == "" {
		trigger = "Not set"
	}
	log.Println("Trigger - Delete file: " + trigger)

	//fmt.Println("Trigger Expired tag: " + cfg.TriggerExpiredTag)

	log.Println("Filebin server starting...")

	router := mux.NewRouter()

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(staticBox.HTTPBox())))

	http.Handle("/", httpInterceptor(router))

	// Accept trailing slashes.
	// Disabling this feature for now since it might not be needed. Try to
	// find some other way of accepting trailing slashes where appropriate
	// instead of globally.
	//router.StrictSlash(true)

	//router.HandleFunc("/admin", reqHandler(api.Admin)).Methods("GET", "HEAD")
	//router.HandleFunc("/api", reqHandler(api.ViewAPI)).Methods("GET", "HEAD")
	//router.HandleFunc("/doc", reqHandler(api.ViewDoc)).Methods("GET", "HEAD")
	router.HandleFunc("/", reqHandler(api.ViewIndex)).Methods("GET", "HEAD")
	router.HandleFunc("/", reqHandler(api.Upload)).Methods("POST")
	router.HandleFunc("/archive/{tag:[A-Za-z0-9_-]+}", reqHandler(api.FetchArchive)).Methods("GET", "HEAD")
	router.HandleFunc("/album/{tag:[A-Za-z0-9_-]+}", reqHandler(api.FetchAlbum)).Methods("GET", "HEAD")
	router.HandleFunc("/{tag:[A-Za-z0-9_-]+}", reqHandler(api.FetchTag)).Methods("GET", "HEAD")
	router.HandleFunc("/{tag:[A-Za-z0-9_-]+}", reqHandler(api.DeleteTag)).Methods("DELETE")
	router.HandleFunc("/{tag:[A-Za-z0-9_-]+}/{filename:.+}", reqHandler(api.FetchFile)).Methods("GET", "HEAD")
	router.HandleFunc("/{tag:[A-Za-z0-9_-]+}/{filename:.+}", reqHandler(api.DeleteFile)).Methods("DELETE")
	router.HandleFunc("/{path:.*}", reqHandler(api.PurgeHandler)).Methods("PURGE")

	//router.HandleFunc("/dashboard{_:/?}", ViewDashboard).Methods("GET", "HEAD")

	//router.HandleFunc("/", ViewIndex).Methods("GET", "HEAD")
	//router.HandleFunc("/upload{_:/?}", RedirectToNewTag).Methods("GET", "HEAD")
	//router.HandleFunc("/upload/{tag:[A-Za-z0-9_-]+}", RedirectOldTag)

	//router.HandleFunc("/{tag:[A-Za-z0-9_-]+}/page/{page:[0-9]+}{_:/?}",
	//    ViewTag).Methods("GET", "HEAD")
	//router.HandleFunc("/{tag:[A-Za-z0-9_-]+}{_:/?}", ViewTag).Methods("GET", "HEAD")

	//router.HandleFunc("/user{_:/?}", user.GetHomePage).Methods("GET")
	//router.HandleFunc("/user/view/{id:[0-9]+}", user.GetViewPage).Methods("GET")
	//router.HandleFunc("/user/{id:[0-9]+}", user.GetViewPage).Methods("GET")

	// Start dispatcher that will handle all background processing
	model.StartDispatcher(cfg.Workers, cfg.CacheInvalidation, WorkQueue, log)

	err := http.ListenAndServe(cfg.Host+":"+strconv.Itoa(cfg.Port), nil)
	if err != nil {
		log.Fatalln(err.Error())
	}
}

func reqHandler(fn func(http.ResponseWriter, *http.Request, config.Configuration, model.Context)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now().UTC()
		reqId := "r-" + generateReqId(5)

		// Populate the context for this request here
		var ctx = model.Context{}
		ctx.TemplateBox = templateBox
		ctx.StaticBox = staticBox
		ctx.Baseurl = cfg.Baseurl
		ctx.WorkQueue = WorkQueue

		// Initialize logger for this request
		ctx.Log = log.New(os.Stdout, reqId+" ", log.LstdFlags)

		ctx.Log.Println(r.Method + " " + r.RequestURI)
		if r.Host != "" {
			ctx.Log.Println("Host: " + r.Host)
		}
		ctx.Log.Println("Remote address: " + r.RemoteAddr)

		// Print X-Forwarded-For since we might be behind some TLS
		// terminator and web cache
		xff := r.Header.Get("X-Forwarded-For")
		if xff != "" {
			ctx.Log.Println("X-Forwarded-For: " + xff)
		}
		ua := r.Header.Get("User-Agent")
		if ua != "" {
			ctx.Log.Println("User-Agent: " + ua)
		}

		fn(w, r, cfg, ctx)

		finishTime := time.Now().UTC()
		elapsedTime := finishTime.Sub(startTime)
		ctx.Log.Println("Response time: " + elapsedTime.String())
	}
}

func httpInterceptor(router http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		router.ServeHTTP(w, r)
	})
}
