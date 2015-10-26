package api

import (
	"crypto/md5"
	"math/rand"
	"os"
	"path"
	"encoding/hex"
	"time"
	"strconv"
	"io"
	"strings"
	"regexp"
	"net/http"
	"path/filepath"
	//"github.com/gorilla/mux"
	"github.com/dustin/go-humanize"
	"github.com/golang/glog"
	"github.com/espebra/filebin/app/config"
	"github.com/espebra/filebin/app/output"
)

type Link struct {
	Rel	string
	Href	string
}

type File struct {
	Filename		string		`json:"filename"`
	Tag			string		`json:"tag"`
	Bytes			uint64		`json:"bytes"`
	BytesReadable		string		`json:"bytes_prefixed"`
	MIME			string		`json:"mime"`
	Verified		bool		`json:"verified"`
	Md5			string		`json:"md5"`
	RemoteAddr		string		`json:"remote-addr"`
	UserAgent		string		`json:"user-agent"`
	CreatedAt		time.Time	`json:"created"`
	CreatedAtReadable	string		`json:"created_relative"`
	ExpiresAt		time.Time	`json:"expires"`
	ExpiresAtReadable	string		`json:"expires_relative"`
	Links			[]Link		`json:"links"`
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func md5sum(filePath string) ([]byte, error) {
    var result []byte
    file, err := os.Open(filePath)
    if err != nil {
        return result, err
    }
    defer file.Close()

    hash := md5.New()
    if _, err := io.Copy(hash, file); err != nil {
        return result, err
    }
   
    return hash.Sum(result), nil
}

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

	if (isDir(tagdir)) {
		glog.Info("The directory " + tagdir + " exists")
	} else {
		glog.Info("The directory " + tagdir + " does not exist. Creating.")
		err := os.Mkdir(tagdir, 0700)
		if err != nil {
			glog.Info("Unable to create directory " + tagdir + ": ", err)
			http.Error(w, "", http.StatusInternalServerError);
			return
		}
	}

	var fpath = filepath.Join(tagdir, filename)
	glog.Info("Writing data to " + fpath)
	fp, err := os.Create(fpath)
	defer fp.Close()
	if err != nil {
		glog.Info("Unable to create file " + fpath + ": ", err)
		http.Error(w, "", http.StatusInternalServerError);
		return
	}

	nBytes, err := io.Copy(fp, r.Body)
	if err != nil {
		glog.Info("Unable to copy request body to file " +
			fpath + ": ", err)
		http.Error(w, "", http.StatusInternalServerError);
		return
	} else {
		glog.Info("Upload complete after " +
			strconv.FormatInt(nBytes, 10) + " bytes")
	}

	var verified = false
	var calculated_md5 = ""
	if b, err := md5sum(fpath); err != nil {
		glog.Info("Error occurred while calculating md5 checksum: ", err)
		http.Error(w, "Upload failed", http.StatusInternalServerError);
		return
	} else {
		calculated_md5 = hex.EncodeToString(b)
		glog.Info("Calculated md5 checksum: " + calculated_md5)
		if md5 == calculated_md5 {
			glog.Info("md5 checksum verified")
			verified = true
		} else {
			if md5 != "" {
				glog.Info("md5 checksum verification failed")
				http.Error(w, "md5 verification failed",
				http.StatusConflict);
				return
			}
		}
	}

	f := File { }
	f.Filename = filename
	f.Tag = tag
	f.Bytes = uint64(nBytes)
	f.Verified = verified
	f.Md5 = calculated_md5
	f.RemoteAddr = r.RemoteAddr
	f.UserAgent = r.Header.Get("User-Agent")

	f.CreatedAt = time.Now().UTC()
	f.ExpiresAt = time.Now().UTC().Add(24 * 7 * 4 * time.Hour)

	f.BytesReadable = humanize.Bytes(f.Bytes)
	f.CreatedAtReadable = humanize.Time(f.CreatedAt)
	f.ExpiresAtReadable = humanize.Time(f.ExpiresAt)


	fileLink := Link {}
	fileLink.Rel = "file"
	fileLink.Href = cfg.Baseurl + "/" + tag + "/" + filename
	f.Links = append(f.Links, fileLink)

	tagLink := Link {}
	tagLink.Rel = "tag"
	tagLink.Href = cfg.Baseurl + "/" + tag
	f.Links = append(f.Links, tagLink)

	buff := make([]byte, 512)
	fp.Seek(0, 0)
	_, err = fp.Read(buff)
	f.MIME = http.DetectContentType(buff)
	defer fp.Close()

	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"

	var status = 200
	output.JSONresponse(w, status, headers, f)
}
