package main

import (
	"fmt"
	"flag"
	"strconv"
	"net/http"
	"time"
	"io"
	"encoding/json"

	"github.com/espebra/filebin/app/config"
	"github.com/espebra/filebin/app/log"
	"github.com/gorilla/mux"
)

func main() {
	flag.StringVar(&config.Global.Filedir, "filedir", config.Global.Filedir,
		"Files directory")

	flag.StringVar(&config.Global.Tempdir, "tempdir", config.Global.Tempdir,
		"Temp directory")

	flag.StringVar(&config.Global.Logfile, "logfile", config.Global.Logfile,
		"Path to log file")

	flag.StringVar(&config.Global.Thumbdir, "thumbdir", config.Global.Thumbdir,
		"Path to thumbnail directory")

	flag.StringVar(&config.Global.Database, "database", config.Global.Database,
		"Path to database file")

	flag.StringVar(&config.Global.Host, "host", config.Global.Host,
		"Listen host")

	flag.IntVar(&config.Global.Port, "port", config.Global.Port, "Listen port")

	flag.IntVar(&config.Global.Readtimeout, "readtimeout",
		config.Global.Readtimeout, "Read timeout in seconds")

	flag.IntVar(&config.Global.Writetimeout, "writetimeout",
		config.Global.Writetimeout, "Write timeout in seconds")

	flag.IntVar(&config.Global.Maxheaderbytes, "maxheaderbytes",
		config.Global.Maxheaderbytes, "Max header bytes.")

	flag.IntVar(&config.Global.Pagination, "pagination",
		config.Global.Pagination, "Files to show per page for pagination.")

	flag.StringVar(&config.Global.GeoIP2, "geoip2",
		config.Global.GeoIP2, "Path to the GeoIP2 database file.")

	flag.BoolVar(&config.Global.Verbose, "verbose", config.Global.Verbose,
		"Verbose stdout.")

	flag.StringVar(&config.Global.TriggerNewTag, "trigger-new-tag",
		config.Global.TriggerNewTag, "Trigger to execute when a new tag is created.")

	flag.StringVar(&config.Global.TriggerUploadedFile, "trigger-uploaded-file",
		config.Global.TriggerUploadedFile,
		"Trigger to execute when a file is uploaded.")

	flag.StringVar(&config.Global.TriggerExpiredTag, "trigger-expired-tag",
		config.Global.TriggerExpiredTag,
		"Trigger to execute when a tag expires.")

	flag.Parse()

	//if (!IsDir(config.Global.Logdir)) {
	//    fmt.Println("The specified log directory is not a directory: ",
	//        config.Global.Logdir)
	//    os.Exit(2)
	//}

	//logfile := filepath.Join(config.Global.Logdir, "filebin.log")
	//f, err := os.OpenFile(logfile, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
	//if err != nil {
	//    fmt.Print("Error opening file: %v", err)
	//}
	//defer f.Close()
	//Error = log.New(f, " ", log.Ldate|log.Ltime|log.Lshortfile)
	//Info = log.New(f, " ", log.Ldate|log.Ltime|log.Lshortfile)
	//Database = log.New(f, " ", log.Ldate|log.Ltime|log.Lshortfile)

	if config.Global.Port < 1 || config.Global.Port > 65535 {
		log.Fatal("Invalid port number, aborting.")
	}

	if config.Global.Readtimeout < 1 || config.Global.Readtimeout > 3600 {
		log.Fatal("Invalid read timeout, aborting.")
	}

	if config.Global.Writetimeout < 1 || config.Global.Writetimeout > 3600 {
		log.Fatal("Invalid write timeout, aborting.")
	}

	if config.Global.Maxheaderbytes < 1 ||
		config.Global.Maxheaderbytes > 2 << 40 {
		log.Fatal("Invalid max header bytes, aborting.")
	}

	//if (!IsDir(config.Global.Tempdir)) {
	//    Info.Fatal("The directory " + config.Global.Tempdir +
	//        " does not exist.")
	//    os.Exit(2)
	//}

	//if (!IsDir(config.Global.Filedir)) {
	//    Info.Fatal("The directory " + config.Global.Filedir +
	//        " does not exist.")
	//    os.Exit(2)
	//}

	//if _, err := os.Stat(config.Global.GeoIP2); err == nil {
	//    gi, err = geoip2.Open(config.Global.GeoIP2)
	//    if err != nil {
	//        Info.Print("Could not open GeoIP2 database ", config.Global.GeoIP2,
	//            ": ", err)
	//    }
	//    defer gi.Close()
	//} else {
	//    Info.Print("GeoIP2 database does not exist: ", config.Global.GeoIP2)
	//}

	if config.Global.Verbose {
		fmt.Println("Host: " + config.Global.Host)
		fmt.Println("Port: " + strconv.Itoa(config.Global.Port))
		fmt.Println("Read timeout: " +
			strconv.Itoa(config.Global.Readtimeout) + " seconds")
		fmt.Println("Write timeout: " +
			strconv.Itoa(config.Global.Writetimeout) + " seconds")
		fmt.Println("Max header bytes: " +
			strconv.Itoa(config.Global.Maxheaderbytes))
		fmt.Println("Files directory: " + config.Global.Filedir)
		fmt.Println("Thumbnail directory: " + config.Global.Thumbdir)
		fmt.Println("Temp directory: " + config.Global.Tempdir)
		fmt.Println("Log file: " + config.Global.Logfile)
		fmt.Println("GeoIP2 database: " + config.Global.GeoIP2)
		fmt.Println("Pagination: " + strconv.Itoa(config.Global.Pagination))
		fmt.Println("Trigger New tag: " + config.Global.TriggerNewTag)
		fmt.Println("Trigger Uploaded file: " + config.Global.TriggerUploadedFile)
		fmt.Println("Trigger Expired tag: " + config.Global.TriggerExpiredTag)
	}

	//err = Setup()
	//if err != nil {
	//    Error.Println("Database setup error: ", err)
	//}

	log.Info("Filebin server starting on " + config.Global.Host + ":" +
		strconv.Itoa(config.Global.Port) + " from directory " +
		config.Global.Filedir)

	router := mux.NewRouter()
	http.Handle("/", httpInterceptor(router))

	//router.HandleFunc("/dashboard{_:/?}", ViewDashboard).Methods("GET", "HEAD")

	//router.HandleFunc("/", ViewIndex).Methods("GET", "HEAD")
	//router.HandleFunc("/upload{_:/?}", RedirectToNewTag).Methods("GET", "HEAD")
	//router.HandleFunc("/upload/{tag:[A-Za-z0-9_-]+}", RedirectOldTag)

	//router.HandleFunc("/{tag:[A-Za-z0-9_-]+}/page/{page:[0-9]+}{_:/?}",
	//    ViewTag).Methods("GET", "HEAD")
	//router.HandleFunc("/{tag:[A-Za-z0-9_-]+}{_:/?}", ViewTag).Methods("GET", "HEAD")
	//router.HandleFunc("/{tag:[A-Za-z0-9_-]+}", UploadToTag).Methods("POST")
	//router.HandleFunc("/{tag:[A-Za-z0-9_-]+}/{filename:.+}", ViewFile).Methods("GET", "HEAD")

	////router.HandleFunc("/user{_:/?}", user.GetHomePage).Methods("GET")
	////router.HandleFunc("/user/view/{id:[0-9]+}", user.GetViewPage).Methods("GET")
	////router.HandleFunc("/user/{id:[0-9]+}", user.GetViewPage).Methods("GET")

	//// CSS and Javascript
	//fileServer := http.StripPrefix("/static/",
	//    http.FileServer(http.Dir("static")))
	//http.Handle("/static/", fileServer)

	err := http.ListenAndServe(config.Global.Host + ":" +
		strconv.Itoa(config.Global.Port), nil)

	if err != nil {
		log.Fatal(err.Error())
	}
}

func httpInterceptor(router http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now().UTC()

		//var req = ParseRequest(r)
		//ReqId := GenerateReqId(16)

		//Info.SetPrefix("Reqid " + ReqId + ": ")
		//Error.SetPrefix("Reqid " + ReqId + ": ")
		//Database.SetPrefix("Reqid " + ReqId + ": ")

		log.Info("Request from " + r.RemoteAddr)
		log.Info(r.Method + " " + r.RequestURI)

		router.ServeHTTP(w, r)

		finishTime := time.Now().UTC()
		elapsedTime := finishTime.Sub(startTime)

		log.Info("Status " + w.Status)
		log.Info("Processing time: " + elapsedTime.String())
	})
}

func JSONresponse(w http.ResponseWriter, status int, h map[string]string, d interface{}) {
	dj, err := json.MarshalIndent(d, "", "    ")
	if err != nil {
		fmt.Print("Unable to convert response to json: ", err)
		http.Error(w, "Failed while generating a response", http.StatusInternalServerError)
		return
	}

	for header, value := range h {
		w.Header().Set(header, value)
	}

	w.WriteHeader(status)
	//log.Info("Status " + strconv.Itoa(status))
	io.WriteString(w, string(dj))
	//Info.Print("Output: ", string(dj))
	fmt.Print("Response status: " + strconv.Itoa(status))
}
