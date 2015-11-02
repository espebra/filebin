package model

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/ioutil"
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
	Tempfile		string		`json:"-"`
	Tag		    	string		`json:"tag"`
	TagDir			string		`json:"-"`

	Bytes			int64		`json:"bytes"`
	MIME			string		`json:"mime"`
	Checksum		string		`json:"checksum"`
	Algorithm		string		`json:"algorithm"`
	Verified		bool		`json:"verified"`
	RemoteAddr		string		`json:"-"`
	UserAgent		string		`json:"-"`
	CreatedAt		time.Time	`json:"created"`
	//ExpiresAt		time.Time	`json:"expires"`
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
		// Use the checksum as the filename
		f.Filename = f.Checksum
		glog.Info("Set checksum as the filename: " + f.Checksum)
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
	path := f.Tempfile

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
	glog.Info("Detected MIME type: " + f.MIME)
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
	path := f.Tempfile
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
			glog.Info("The provided checksum is not correct: " + s)
			err = errors.New("Checksum " + s + " did not match " +
				f.Checksum)
		}
	}
	glog.Info("Checksum is ", f.Checksum)
	return err
}

func (f *File) WriteTempfile(d io.Reader, tempdir string) error {
	fp, err := ioutil.TempFile(tempdir, "upload")
	if err != nil {
		return err
	}
	glog.Info("Writing data to " + fp.Name())

	f.Bytes, err = io.Copy(fp, d)
	if err != nil {
		return err
	}
	glog.Info("Upload complete after " + strconv.FormatInt(f.Bytes, 10) +
		" bytes")

	fp.Sync()

	// Store the tempfile path for later
	f.Tempfile = fp.Name()
	defer fp.Close()
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

func (f *File) Publish() error {
	err := CopyFile(f.Tempfile, filepath.Join(f.TagDir, f.Filename))
	return err
}

func (f *File) ClearTemp() error {
	err := os.Remove(f.Tempfile)
	if err != nil {
		glog.Error("Unable to remove tempfile ", f.Tempfile, ": ", err)
		return err
	}
	glog.Info("Removed tempfile: ", f.Tempfile)
	return nil
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


// http://stackoverflow.com/a/21067803
// CopyFile copies a file from src to dst. If src and dst files exist, and are
// the same, then return success. Otherise, attempt to create a hard link
// between the two files. If that fail, copy the file contents from src to dst.
func CopyFile(src, dst string) (err error) {
	sfi, err := os.Stat(src)
	if err != nil {
		return
	}
	if !sfi.Mode().IsRegular() {
		// cannot copy non-regular files (e.g., directories,
		// symlinks, devices, etc.)
		return errors.New("CopyFile: non-regular source file " + sfi.Name() + ": " + sfi.Mode().String())
	}
	dfi, err := os.Stat(dst)
	if err != nil {
		if !os.IsNotExist(err) {
			return
		}
	} else {
		if !(dfi.Mode().IsRegular()) {
			return errors.New("CopyFile: non-regular destination file " + dfi.Name() + ": " + dfi.Mode().String())
		}
		if os.SameFile(sfi, dfi) {
			return
		}
	}
	if err = os.Link(src, dst); err == nil {
		return
	}
	err = copyFileContents(src, dst)
	return err
}

// copyFileContents copies the contents of the file named src to the file named
// by dst. The file will be created if it does not already exist. If the
// destination file exists, all it's contents will be replaced by the contents
// of the source file.
func copyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	_, err = io.Copy(out, in)
	if err != nil {
		return
	}
	err = out.Sync()
	return err
}
