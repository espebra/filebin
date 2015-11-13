package main

import (
	"os"
	"os/signal"
	"syscall"
	"fmt"
	"flag"
	"time"
	"strconv"
	"net/http"
	"math/rand"

	"github.com/gorilla/mux"
	"github.com/golang/glog"
	"github.com/GeertJohan/go.rice"

	"github.com/espebra/filebin/app/config"
	"github.com/espebra/filebin/app/model"
	"github.com/espebra/filebin/app/api"
)

var cfg = config.Global
var githash = "No githash provided"
var buildstamp = "No buildstamp provided"

var staticBox *rice.Box
var templateBox *rice.Box

func generateReqId(n int) string {
        var letters = []rune("abcdefghijklmnopqrstuvwxyz0123456789")
        b := make([]rune, n)
        for i := range b {
                b[i] = letters[rand.Intn(len(letters))]
        }
        return string(b)
}

func teardown() {
    glog.Flush()
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	go func() {
		<-c
		teardown()
		os.Exit(1)
	}()

	defer glog.Flush()

	flag.StringVar(&cfg.Baseurl, "baseurl",
		cfg.Baseurl, "Baseurl used when generating links.")

	flag.StringVar(&cfg.Filedir, "filedir",
		cfg.Filedir, "Directory to store uploaded files.")

	flag.StringVar(&cfg.Tempdir, "tempdir",
		cfg.Tempdir, "Directory to temporarily store files during upload.")

	flag.StringVar(&cfg.Logdir, "logdir",
		cfg.Logdir, "Directory to write log files to.")

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

	//flag.IntVar(&cfg.Pagination, "pagination",
	//	cfg.Pagination,
	//	"Files to show per page for pagination.")

	//flag.StringVar(&cfg.GeoIP2, "geoip2",
	//	cfg.GeoIP2, "Path to the GeoIP2 database file.")

	flag.Int64Var(&cfg.Expiration, "expiration",
		cfg.Expiration, "Tag expiration time in seconds after the last modification.")

	flag.BoolVar(&cfg.Verbose, "verbose",
		cfg.Verbose, "Verbose output.")

//	flag.StringVar(&cfg.TriggerNewTag, "trigger-new-tag",
//		cfg.TriggerNewTag,
//		"Trigger to execute when a new tag is created.")

	flag.StringVar(&cfg.TriggerUploadedFile,
		"trigger-uploaded-file",
		cfg.TriggerUploadedFile,
		"Command to execute when a file is uploaded.")

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

	flag.Lookup("logtostderr").Value.Set("false")
	flag.Lookup("log_dir").Value.Set(cfg.Logdir)
	flag.Lookup("log_dir").Value.Set(cfg.Logdir)
	flag.Lookup("v").Value.Set("100")

	if cfg.Port < 1 || cfg.Port > 65535 {
		glog.Fatal("Invalid port number, aborting.")
	}

	if cfg.Readtimeout < 1 || cfg.Readtimeout > 3600 {
		glog.Fatal("Invalid read timeout, aborting.")
	}

	if cfg.Writetimeout < 1 || cfg.Writetimeout > 3600 {
		glog.Fatal("Invalid write timeout, aborting.")
	}

	if cfg.Maxheaderbytes < 1 ||
		cfg.Maxheaderbytes > 2 << 40 {
		glog.Fatal("Invalid max header bytes, aborting.")
	}

	//if (!IsDir(cfg.Tempdir)) {
	//    Info.Fatal("The directory " + cfg.Tempdir +
	//        " does not exist.")
	//    os.Exit(2)
	//}

	//if (!IsDir(cfg.Filedir)) {
	//    glog.Fatal("The directory " + cfg.Filedir +
	//        " does not exist.")
	//}

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
	// Initialize boxes
	staticBox = rice.MustFindBox("static")
	templateBox = rice.MustFindBox("templates")

	if cfg.Verbose {
		fmt.Println("Listen host: " + cfg.Host)
		fmt.Println("Listen port: " + strconv.Itoa(cfg.Port))
		fmt.Println("Read timeout: " +
			strconv.Itoa(cfg.Readtimeout) + " seconds")
		fmt.Println("Write timeout: " +
			strconv.Itoa(cfg.Writetimeout) + " seconds")
		fmt.Println("Max header size: " +
			strconv.Itoa(cfg.Maxheaderbytes) + " bytes")
		fmt.Println("Expiration time: " + strconv.FormatInt(cfg.Expiration, 10) +
			" seconds")
		fmt.Println("Files directory: " + cfg.Filedir)
		//fmt.Println("Thumbnail directory: " + cfg.Thumbdir)
		fmt.Println("Temp directory: " + cfg.Tempdir)
		fmt.Println("Log directory: " + cfg.Logdir)
		//fmt.Println("GeoIP2 database: " + cfg.GeoIP2)
		//fmt.Println("Pagination: " + strconv.Itoa(cfg.Pagination))
		fmt.Println("Baseurl: " + cfg.Baseurl)
		//fmt.Println("Trigger New tag: " + cfg.TriggerNewTag)

                var trigger = cfg.TriggerUploadedFile
                if trigger == "" {
                    trigger = "Not set"
                }
		fmt.Println("Trigger - Uploaded file: " + trigger)

		//fmt.Println("Trigger Expired tag: " + cfg.TriggerExpiredTag)
	}

	//err = Setup()
	//if err != nil {
	//    Error.Println("Database setup error: ", err)
	//}

	glog.Info("Filebin server starting on " + cfg.Host + ":" +
		strconv.Itoa(cfg.Port) + " from directory " +
		cfg.Filedir)

	router := mux.NewRouter()

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(staticBox.HTTPBox())))

	http.Handle("/", httpInterceptor(router))

	router.HandleFunc("/", reqHandler(api.ViewIndex)).Methods("GET", "HEAD")
	router.HandleFunc("/", reqHandler(api.Upload)).Methods("POST")
	router.HandleFunc("/{tag:[A-Za-z0-9_-]+}", reqHandler(api.FetchTag)).Methods("GET", "HEAD")
	router.HandleFunc("/{tag:[A-Za-z0-9_-]+}/{filename:.+}", reqHandler(api.FetchFile)).Methods("GET", "HEAD")
	router.HandleFunc("/{tag:[A-Za-z0-9_-]+}/{filename:.+}", reqHandler(api.DeleteFile)).Methods("DELETE")

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

	//// CSS and Javascript
	//fileServer := http.StripPrefix("/static/",
	//    http.FileServer(http.Dir("static")))
	//http.Handle("/static/", fileServer)

	err := http.ListenAndServe(cfg.Host + ":" +
		strconv.Itoa(cfg.Port), nil)

	if err != nil {
		glog.Fatal(err.Error())
	}
}

func reqHandler(fn func (http.ResponseWriter, *http.Request, config.Configuration, model.Context)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now().UTC()

		//var req = ParseRequest(r)

		glog.Info("Request from " + r.RemoteAddr)
		glog.Info(r.Method + " " + r.RequestURI)
		//ReqId := generateReqId(16)
		//glog.Info("ReqId:", ReqId)

		// Populate the context for this request here
		var ctx = model.Context {}
		ctx.TemplateBox = templateBox
		ctx.StaticBox = staticBox

		fn(w, r, cfg, ctx)

		finishTime := time.Now().UTC()
		elapsedTime := finishTime.Sub(startTime)

		//log.Info("Status " + w.Status)
		glog.Info("Processing time: " + elapsedTime.String())
		glog.Flush()
	}
}

func httpInterceptor(router http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		router.ServeHTTP(w, r)
	})
}


