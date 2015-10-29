package api

import (
	"crypto/sha256"
	"math/rand"
	"errors"
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
	//"github.com/dustin/go-humanize"
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
	TagPath			string		`json:"-"`

	Bytes			int64		`json:"bytes"`
	//BytesReadable		string		`json:"bytes_prefixed"`
	MIME			string		`json:"mime"`
	Verified		bool		`json:"verified"`
	SHA256			string		`json:"sha256"`
	RemoteAddr		string		`json:"remote-addr"`
	UserAgent		string		`json:"-"`
	CreatedAt		time.Time	`json:"created"`
	//CreatedAtReadable	string		`json:"created_relative"`
	ExpiresAt		time.Time	`json:"expires"`
	//ExpiresAtReadable	string		`json:"expires_relative"`
	Links			[]Link		`json:"links"`
}

func (f *File) SetFilename(s string) {
	var sanitized = path.Base(path.Clean(s))

	// Remove any trailing space to avoid ending on -
	sanitized = strings.Trim(sanitized, " ")

	// Remove all but valid chars
	var valid = regexp.MustCompile("[^A-Za-z0-9-_=,. ]")
	sanitized = valid.ReplaceAllString(sanitized, "_")

	if sanitized == "" {
		// Generate filename if not provided
		f.Filename = randomString(16)
		glog.Info("Generated filename: " + f.Filename)
	} else {
		f.Filename = sanitized
	}
}

func (f *File) SetTag(s string) error {
	var err error
	if s == "" {
		// Generate tag if not provided
		f.Tag = randomString(16)
		glog.Info("Generated tag: " + f.Tag)
	} else {
		if validTag(s) {
			f.Tag = s
		} else {
			err = errors.New("Invalid tag specified. It contains " +
				"illegal characters or is too short.")
		}
	}
	return err
}

func (f *File) VerifySHA256(s string) error {
	var err error
	path := filepath.Join(f.TagPath, f.Filename)
	if f.SHA256 == "" {
		f.SHA256, err = sha256sum(path)
	}
	if s == "" {
		f.Verified = false
	} else {
		if f.SHA256 == s {
			f.Verified = true
		} else {
			err = errors.New("Checksum " + s + " did not match " +
				f.SHA256)
		}
	}
	return err
}

func (f *File) EnsureTagDirectoryExists() error {
	var err error
	if !isDir(f.TagPath) {
		glog.Info("The directory " + f.TagPath + " does not exist. " +
			"Creating.")
		err = os.Mkdir(f.TagPath, 0700)
	}
	return err
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
	if err != nil {
		glog.Error("Trigger command failed: ", err)
	}
	return err
}

func sha256sum(filePath string) (string, error) {
    var result []byte
    file, err := os.Open(filePath)
    if err != nil {
        return "", err
    }
    defer file.Close()

    hash := sha256.New()
    if _, err := io.Copy(hash, file); err != nil {
        return "", err
    }
   
    return hex.EncodeToString(hash.Sum(result)), nil
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

func validTag(tag string) (bool) {
	var validTag = regexp.MustCompile("^[a-zA-Z0-9-_]{8,}$")
	if validTag.MatchString(tag) {
		return true
	} else {
		return false
	}
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
	var err error

	f := File { }
	f.SetFilename(r.Header.Get("filename"))
	err = f.SetTag(r.Header.Get("tag"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest);
		return
	}

	f.TagPath = filepath.Join(cfg.Filedir, f.Tag)

	//var sha256 = r.Header.Get("content-sha256")
	//if (sha256 != "") {
	//	glog.Info("Provided SHA256 checksum: ", sha256)
	//}

	err = f.EnsureTagDirectoryExists()
	if err != nil {
		glog.Error("Unable to create tag directory", f.TagPath)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError);
		return
	}

	f.Bytes, err = writeToFile(filepath.Join(f.TagPath, f.Filename), r.Body)
	if err != nil {
		glog.Info("Unable to write file " + filepath.Join(f.TagPath, f.Filename) + ":", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError);
		return
	}

	err = f.VerifySHA256(r.Header.Get("content-sha256"))
	if err != nil {
		http.Error(w, "Checksum did not match", http.StatusConflict);
		return
	}

	f.RemoteAddr = r.RemoteAddr
	f.UserAgent = r.Header.Get("User-Agent")
	f.CreatedAt = time.Now().UTC()
	f.ExpiresAt = time.Now().UTC().Add(24 * 7 * 4 * time.Hour)
	//f.BytesReadable = humanize.Bytes(uint64(f.Bytes))
	//f.CreatedAtReadable = humanize.Time(f.CreatedAt)
	//f.ExpiresAtReadable = humanize.Time(f.ExpiresAt)
	f.MIME, err = detectMIME(filepath.Join(f.TagPath, f.Filename))
	if err != nil {
		glog.Info("Unable to detect MIME for file " + filepath.Join(f.TagPath, f.Filename) + ":", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError);
		return
	}

	fileLink := Link {}
	fileLink.Rel = "file"
	fileLink.Href = cfg.Baseurl + "/" + f.Tag + "/" + f.Filename
	f.Links = append(f.Links, fileLink)

	tagLink := Link {}
	tagLink.Rel = "tag"
	tagLink.Href = cfg.Baseurl + "/" + f.Tag
	f.Links = append(f.Links, tagLink)

	if cfg.TriggerUploadedFile != "" {
		triggerUploadedFileHandler(cfg.TriggerUploadedFile, f.Tag, f.Filename)
	}

	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"

	var status = 200
	output.JSONresponse(w, status, headers, f)
}

//func FetchFile(w http.ResponseWriter, r *http.Request, cfg config.Configuration) {
//    params := mux.Vars(r)
//    tag := params["tag"]
//    filename := params["filename"]
//
//    if (!validTag(tag)) {
//        http.Error(w,"Invalid tag specified. It contains illegal characters or is too short.", 400)
//        return
//    }
//
//    filename = sanitizeFilename(filename)
//    if (filename == "") {
//        http.Error(w, "Filename invalid or not set.", 400)
//        return
//    }
//
//    path := filepath.Join(cfg.Filedir, tag, filename)
//
//    w.Header().Set("Cache-Control", "max-age: 60")
//    http.ServeFile(w, r, path)
//}
