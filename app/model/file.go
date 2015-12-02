package model

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

        "github.com/dustin/go-humanize"
)

type File struct {
	Filename		string		`json:"filename"`
	Tag			string		`json:"tag"`
	TagDir			string		`json:"-"`
	Bytes			int64		`json:"bytes"`
	BytesReadable		string		`json:"-"`
	MIME			string		`json:"mime"`
	CreatedReadable		string		`json:"created"`
	CreatedAt		time.Time	`json:"-"`
	Links			[]Link		`json:"links"`
	Checksum		string		`json:"checksum"`
	Algorithm		string		`json:"algorithm"`
	Verified		bool		`json:"verified"`
	RemoteAddr		string		`json:"-"`
	UserAgent		string		`json:"-"`
	Tempfile		string		`json:"-"`
}

func (f *File) SetTag(s string) error {
        validTag := regexp.MustCompile("^[a-zA-Z0-9-_]{8,}$")
        if validTag.MatchString(s) == false {
                return errors.New("Invalid tag specified. It contains " +
                        "illegal characters or is too short")
        }
        f.Tag = s
        return nil
}

func (f *File) SetTagDir(filedir string) error {
	if f.Tag == "" {
		return errors.New("Tag not set.")
	}
        f.TagDir = filepath.Join(filedir, f.Tag)
	return nil
}

func (f *File) SetFilename(s string) error {
	// Remove all but valid chars
	var valid = regexp.MustCompile("[^A-Za-z0-9-_=,.]")
	var safe = valid.ReplaceAllString(s, "_")
	if safe == "" {
		return errors.New("Invalid filename specified. It contains " +
			"illegal characters or is too short.")
	}

	// Reject illegal filenames
	switch safe {
		case ".", "..":
			return errors.New("Invalid filename specified.")
	}

	// Set filename to the safe variant
	f.Filename = safe

	return nil
}

func (f *File) DetectMIME() error {
	if f.TagDir == "" {
		return errors.New("TagDir not set.")
	}
	if f.Filename == "" {
		return errors.New("Filename not set.")
	}
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

func (f *File) EnsureTagDirectoryExists() error {
	var err error
	if !isDir(f.TagDir) {
		err = os.Mkdir(f.TagDir, 0700)
	}
	return err
}

func (f *File) Exists() bool {
	if f.TagDir == "" {
		return false
	}

	if f.Filename == "" {
		return false
	}

	path := filepath.Join(f.TagDir, f.Filename)
	if !isFile(path) {
		return false
	}
	return true
}

func (f *File) Info() error {
	if isDir(f.TagDir) == false {
		return errors.New("Tag does not exist.")
	}
	
	path := filepath.Join(f.TagDir, f.Filename)
	i, err := os.Lstat(path)
	if err != nil {
		return err
	}
	f.CreatedAt = i.ModTime().UTC()
	f.CreatedReadable = humanize.Time(f.CreatedAt)
	f.Bytes = i.Size()
	f.BytesReadable = humanize.Bytes(uint64(f.Bytes))
	
	//i, err = os.Lstat(f.TagDir)
	//if err != nil {
	//	return err
	//}
	//f.ExpiresAt = i.ModTime().UTC().Add(time.Duration(expiration) * time.Second)
	//f.ExpiresReadable = humanize.Time(f.ExpiresAt)
	return nil
}

func (f *File) Remove() error {
	if f.TagDir == "" {
		return errors.New("Tag dir is not set")
	}

	if !isDir(f.TagDir) {
		return errors.New("Tag dir does not exist")
	}

	path := filepath.Join(f.TagDir, f.Filename)
	
	err := os.Remove(path)
	return err
}

func (f *File) WriteTempfile(d io.Reader, tempdir string) error {
	fp, err := ioutil.TempFile(tempdir, "upload")
	if err != nil {
		return err
	}
	f.Tempfile = fp.Name()

	f.Bytes, err = io.Copy(fp, d)
	if err != nil {
		return err
	}

	fp.Sync()

	// Store the tempfile path for later
	defer fp.Close()
	return nil
}

func (f *File) calculateSHA256(path string) error {
	var err error
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
	return nil
}

func (f *File) VerifySHA256(s string) error {
	var err error
	if f.Checksum == "" {
		err = f.calculateSHA256(f.Tempfile)
		if err != nil {
			return err
		}
	}

	if s == "" {
		f.Verified = false
		return nil
	}

	if f.Checksum == s {
		f.Verified = true
		return nil
	}

	return errors.New("Checksum " + s + " did not match " + f.Checksum)
}

func (f *File) Publish() error {
	err := CopyFile(f.Tempfile, filepath.Join(f.TagDir, f.Filename))
	return err
}

func (f *File) ClearTemp() error {
	err := os.Remove(f.Tempfile)
	return err
}

//func (f *File) GenerateTag() error {
//        var tag = randomString(16)
//        err := f.SetTag(tag)
//        return err
//}

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

func isFile(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	if fi.IsDir() {
		return false
	} else {
		return true
	}
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
