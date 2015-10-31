package model

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"math/rand"
	"net/http"
	"os"
	//"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
)

type Link struct {
	Rel	string
	Href	string
}

type File struct {
	Filename		string		`json:"filename"`
	Tag			string		`json:"tag"`
	TagDir			string		`json:"-"`

	Bytes			int64		`json:"bytes"`
	//BytesReadable		string		`json:"bytes_prefixed"`
	MIME			string		`json:"mime"`
	Checksum		string		`json:"checksum"`
	Algorithm		string		`json:"algorithm"`
	Verified		bool		`json:"verified"`
	RemoteAddr		string		`json:"-"`
	UserAgent		string		`json:"-"`
	CreatedAt		time.Time	`json:"created"`
	//CreatedAtReadable	string		`json:"created_relative"`
	ExpiresAt		time.Time	`json:"expires"`
	//ExpiresAtReadable	string		`json:"expires_relative"`
	Links			[]Link		`json:"links"`
}

func (f *File) SetFilename(s string) {
	var sanitized string
	//var sanitized = path.Base(path.Clean(s))

	// Remove any trailing space to avoid ending on -
	sanitized = strings.Trim(s, " ")

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

func (f *File) GenerateLinks(baseurl string) {
	fileLink := Link {}
	fileLink.Rel = "file"
	fileLink.Href = baseurl + "/" + f.Tag + "/" + f.Filename
	f.Links = append(f.Links, fileLink)

	tagLink := Link {}
	tagLink.Rel = "tag"
	tagLink.Href = baseurl + "/" + f.Tag
	f.Links = append(f.Links, tagLink)
}

func (f *File) DetectMIME() error {
	var err error
	path := filepath.Join(f.TagDir, f.Filename)

	fp, err := os.Open(path)
	if err != nil {
		return err
	}
	defer fp.Close()
	buffer := make([]byte, 512)
	_, err = fp.Seek(0, 0)
	if err != nil {
		return err
	}
	_, err = fp.Read(buffer)
	if err != nil {
		return err
	}
	f.MIME = http.DetectContentType(buffer)
	return nil
}

func (f *File) SetTag(s string) error {
	var err error
	if s == "" {
		// Generate tag if not provided
		f.Tag = randomString(16)
		glog.Info("Generated tag: " + f.Tag)
	} else {
		var validTag = regexp.MustCompile("^[a-zA-Z0-9-_]{8,}$")
		if validTag.MatchString(s) {
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
	path := filepath.Join(f.TagDir, f.Filename)
	if f.Checksum == "" {
		var result []byte
    		fp, err := os.Open(path)
    		if err != nil {
    		    return err
    		}
    		defer fp.Close()

    		hash := sha256.New()
    		_, err = io.Copy(hash, fp)
		if err != nil {
    		    return err
    		}
		f.Checksum = hex.EncodeToString(hash.Sum(result))
		f.Algorithm = "SHA256"
	}
	if s == "" {
		f.Verified = false
	} else {
		if f.Checksum == s {
			f.Verified = true
		} else {
			err = errors.New("Checksum " + s + " did not match " +
				f.Checksum)
		}
	}
	return err
}

func (f *File) WriteFile(d io.Reader) error {
	path := filepath.Join(f.TagDir, f.Filename)
	glog.Info("Writing data to " + path)
	fp, err := os.Create(path)
	defer fp.Close()
	if err != nil {
		return err
	}

	f.Bytes, err = io.Copy(fp, d)
	if err != nil {
		return err
	}
	glog.Info("Upload complete after " + strconv.FormatInt(f.Bytes, 10) +
		" bytes")
	return nil
}

func (f *File) EnsureTagDirectoryExists() error {
	var err error
	if !isDir(f.TagDir) {
		glog.Info("The directory " + f.TagDir + " does not exist. " +
			"Creating.")
		err = os.Mkdir(f.TagDir, 0700)
	}
	return err
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

