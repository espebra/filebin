package api

import (
	"math/rand"
	"path"
	"strings"
	"regexp"
	"net/http"
	"path/filepath"
	//"github.com/gorilla/mux"
	"github.com/golang/glog"
	"github.com/espebra/filebin/app/config"
)

func randomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyz0123456789")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func sanitizeFilename(s string) string {
	var sanitized = path.Base(path.Clean(s))

	// Remove any trailing space to avoid ending on -
	sanitized = strings.Trim(sanitized, " ")

	// Remove all but valid chars
	var valid = regexp.MustCompile("[^A-Za-z0-9-_=,. ]")
	sanitized = valid.ReplaceAllString(sanitized, "_")

	return sanitized
}

func validTag(tag string) (bool) {
	var validTag = regexp.MustCompile("^[a-zA-Z0-9-_]{8,}$")
	if validTag.MatchString(tag) {
		return true
	} else {
		return false
	}
}

func Upload(w http.ResponseWriter, r *http.Request, cfg config.Configuration) {
	//params := mux.Vars(r)

	filename := r.Header.Get("filename")
	if (filename != "") {
		filename = sanitizeFilename(filename)
	}

	if (filename == "") {
		http.Error(w, "Filename invalid or not set.", 400)
		return
	}

	tag := r.Header.Get("tag")
	if tag != "" {
		if (!validTag(tag)) {
			http.Error(w,"Invalid tag specified. It contains illegal characters or is too short.", 400)
			return
		}
	} else {
		tag = randomString(16)
		glog.Info("Generated tag: " + tag)
	}

	md5 := r.Header.Get("content-md5")
	if (md5 == "") {
		glog.Info("md5 checksum was not provided")
	} else {
		glog.Info("Provided md5 checksum: ", md5)
	}

	var tagdir = filepath.Join(cfg.Filedir, tag)

	//if (IsDir(tagdir)) {
	//	Info.Print("The directory " + tagdir + " exists")
	//} else {
	//	Info.Print("The directory " + tagdir + " does not exist. Creating.")
	//	err := os.Mkdir(tagdir, 0700)
	//	if err != nil {
	//		Info.Print("Unable to create directory " + tagdir + ": ", err)
	//		http.Error(w, "", http.StatusInternalServerError);
	//		return
	//	}
	//}

	http.Error(w,"Tag " + tag + ", filename: " + filename + ", tagdir: " + tagdir, 200)
	return
}
