package api

import (
	"crypto/sha256"
	"math/rand"
	"os"
	"os/exec"
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
	SHA256			string		`json:"sha256"`
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

func triggerNewTagHandler(c string, tag string) error {
	glog.Info("Executing trigger-new-tag: " + c)
	cmd := exec.Command(c, tag)
	err := cmdHandler(cmd)
	return err
}

func triggerUploadedFileHandler(c string, tag string, filename string) error {
	glog.Info("Executing trigger-uploaded-file: " + c)
	cmd := exec.Command(c, tag, filename)
	err := cmdHandler(cmd)
	return err
}

func triggerExpiredTagHandler(c string, tag string) error {
	glog.Info("Executing trigger-expired-tag: " + c)
	cmd := exec.Command(c, tag)
	err := cmdHandler(cmd)
	return err
}

func cmdHandler(cmd *exec.Cmd) error {
	err := cmd.Start()
	if err == nil {
		glog.Info("Trigger command executed successfully")
	} else {
		glog.Error("Trigger command failed: ", err)
	}
	return err
}

func sha256sum(filePath string) ([]byte, error) {
    var result []byte
    file, err := os.Open(filePath)
    if err != nil {
        return result, err
    }
    defer file.Close()

    hash := sha256.New()
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

func ensureDirectoryExists(tagdir string) error {
	var err error
	if (isDir(tagdir)) {
		glog.Info("The directory " + tagdir + " exists")
	} else {
		glog.Info("The directory " + tagdir + " does not exist. Creating.")
		err = os.Mkdir(tagdir, 0700)
	}
	return err
}

func writeToFile(path string, body io.Reader) (int64, error) {
	glog.Info("Writing data to " + path)
	fp, err := os.Create(path)
	defer fp.Close()
	if err != nil {
		return 0, err
	}

	nBytes, err := io.Copy(fp, body)
	if err != nil {
		return nBytes, err
	} else {
		glog.Info("Upload complete after " +
			strconv.FormatInt(nBytes, 10) + " bytes")
	}
	return nBytes, nil
}

func detectMIME(path string) (string, error) {
	var err error
	fp, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer fp.Close()
	buff := make([]byte, 512)
	_, err = fp.Seek(0, 0)
	if err != nil {
		return "", err
	}
	_, err = fp.Read(buff)
	if err != nil {
		return "", err
	}
	mime := http.DetectContentType(buff)
	return mime, err
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

	sha256 := r.Header.Get("content-sha256")
	if (sha256 == "") {
		glog.Info("sha256 checksum was not provided")
	} else {
		glog.Info("Provided sha256 checksum: ", sha256)
	}

	var tagdir = filepath.Join(cfg.Filedir, tag)
	err := ensureDirectoryExists(tagdir)
	if err != nil {
		glog.Info("Unable to directory " + tagdir + ": ", err)
		http.Error(w, "", http.StatusInternalServerError);
		return
	}

	var fpath = filepath.Join(tagdir, filename)
	nBytes, err := writeToFile(fpath, r.Body)
	if err != nil {
		glog.Info("Unable to write file " + fpath + ":", err)
		http.Error(w, "", http.StatusInternalServerError);
		return
	}

	var verified = false
	var calculated_sha256 = ""
	if b, err := sha256sum(fpath); err != nil {
		glog.Info("Error occurred while calculating sha256 checksum: ", err)
		http.Error(w, "Upload failed", http.StatusInternalServerError);
		return
	} else {
		calculated_sha256 = hex.EncodeToString(b)
		glog.Info("Calculated sha256 checksum: " + calculated_sha256)
		if sha256 == calculated_sha256 {
			glog.Info("sha256 checksum verified")
			verified = true
		} else {
			if sha256 != "" {
				glog.Info("sha256 checksum verification failed")
				http.Error(w, "sha256 verification failed",
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
	f.SHA256 = calculated_sha256
	f.RemoteAddr = r.RemoteAddr
	f.UserAgent = r.Header.Get("User-Agent")
	f.CreatedAt = time.Now().UTC()
	f.ExpiresAt = time.Now().UTC().Add(24 * 7 * 4 * time.Hour)
	f.BytesReadable = humanize.Bytes(f.Bytes)
	f.CreatedAtReadable = humanize.Time(f.CreatedAt)
	f.ExpiresAtReadable = humanize.Time(f.ExpiresAt)
	f.MIME, err = detectMIME(fpath)
	if err != nil {
		glog.Info("Unable to detect MIME for file " + fpath + ":", err)
		http.Error(w, "", http.StatusInternalServerError);
		return
	}

	fileLink := Link {}
	fileLink.Rel = "file"
	fileLink.Href = cfg.Baseurl + "/" + tag + "/" + filename
	f.Links = append(f.Links, fileLink)

	tagLink := Link {}
	tagLink.Rel = "tag"
	tagLink.Href = cfg.Baseurl + "/" + tag
	f.Links = append(f.Links, tagLink)

	triggerUploadedFileHandler(cfg.TriggerUploadedFile, tag, filename)

	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"

	var status = 200
	output.JSONresponse(w, status, headers, f)
}
