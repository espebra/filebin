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
	"strconv"
	"time"

	"github.com/disintegration/imaging"
	"github.com/dustin/go-humanize"
	"github.com/rwcarlsen/goexif/exif"
)

type File struct {
	Filename        string      `json:"filename"`
	Tag             string      `json:"tag"`
	TagDir          string      `json:"-"`
	Bytes           int64       `json:"bytes"`
	BytesReadable   string      `json:"-"`
	MIME            string      `json:"mime"`
	CreatedReadable string      `json:"-"`
	CreatedAt       time.Time   `json:"created"`
	Links           []Link      `json:"links"`
	Checksum        string      `json:"checksum,omitempty"`
	Algorithm       string      `json:"algorithm,omitempty"`
	Verified        bool        `json:"verified"`
	RemoteAddr      string      `json:"-"`
	UserAgent       string      `json:"-"`
	Tempfile        string      `json:"-"`

	// Image specific attributes
	DateTime  time.Time  `json:"datetime,omitempty"`
	DateTimeReadable  string  `json:"-"`
	Longitude float64    `json:"longitude,omitempty"`
	Latitude  float64    `json:"latitude,omitempty"`
	Altitude  string     `json:"altitude,omitempty"`
	Exif      *exif.Exif `json:"-"`
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
	var err error
	if f.TagDir == "" {
		return errors.New("TagDir not set.")
	}

	fpath := filepath.Join(f.TagDir, f.Filename)
	if f.Tempfile != "" {
		fpath = filepath.Join(f.Tempfile)
	}

	fp, err := os.Open(fpath)
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

func (f *File) MediaType() string {
	s := regexp.MustCompile("/").Split(f.MIME, 2)
	if len(s) > 0 {
		return s[0]
	}
	return ""
}

func (f *File) GenerateLinks(baseurl string) {
	fileLink := Link{}
	fileLink.Rel = "file"
	fileLink.Href = baseurl + "/" + f.Tag + "/" + f.Filename
	f.Links = append(f.Links, fileLink)

	if f.ImageExists(75, 75) {
		thumbLink := Link{}
		thumbLink.Rel = "thumbnail"
		thumbLink.Href = baseurl + "/" + f.Tag + "/" + f.Filename + "?width=75&height=75"
		f.Links = append(f.Links, thumbLink)
	}

	if f.ImageExists(1140, 0) {
		albumItemLink := Link{}
		albumItemLink.Rel = "albumitem"
		albumItemLink.Href = baseurl + "/" + f.Tag + "/" + f.Filename + "?width=1140"
		f.Links = append(f.Links, albumItemLink)
	}

	tagLink := Link{}
	tagLink.Rel = "tag"
	tagLink.Href = baseurl + "/" + f.Tag
	f.Links = append(f.Links, tagLink)

	albumLink := Link{}
	albumLink.Rel = "album"
	albumLink.Href = baseurl + "/album/" + f.Tag
	f.Links = append(f.Links, albumLink)

	archiveLink := Link{}
	archiveLink.Rel = "archive"
	archiveLink.Href = baseurl + "/archive/" + f.Tag
	f.Links = append(f.Links, archiveLink)
}

func (f *File) EnsureTagDirectoryExists() error {
	var err error

	// Tag directory
	if !isDir(f.TagDir) {
		err = os.Mkdir(f.TagDir, 0700)
		if err != nil {
			return err
		}
	}

	// Tag specific cache directory
	cpath := filepath.Join(f.TagDir, ".cache")
	if !isDir(cpath) {
		err = os.Mkdir(cpath, 0700)
		if err != nil {
			return err
		}
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
	if isFile(path) {
		return true
	}
	return false
}

func (f *File) ImageExists(width int, height int) bool {
	path := f.ImagePath(width, height)
	if isFile(path) {
		return true
	}
	return false
}

func (f *File) StatInfo() error {
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
	defer fp.Close()
	if err != nil {
		return err
	}
	f.Tempfile = fp.Name()

	f.Bytes, err = io.Copy(fp, d)
	if err != nil {
		return err
	}

	fp.Sync()
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

func (f *File) ParseExif() error {
	fpath := filepath.Join(f.TagDir, f.Filename)
	if f.Tempfile != "" {
		fpath = f.Tempfile
	}
	fp, err := os.Open(fpath)
	defer fp.Close()
	if err != nil {
		return err
	}

	f.Exif, err = exif.Decode(fp)
	if err != nil {
		return err
	}

	f.DateTime, err = f.Exif.DateTime()
	if err != nil {
		/// XXX: Log
	} else {
		f.DateTimeReadable = humanize.Time(f.DateTime)
	}

	f.Latitude, f.Longitude, err = f.Exif.LatLong()
	if err != nil {
		/// XXX: Log
	}

	return nil
}

func (f *File) Publish() error {
	err := CopyFile(f.Tempfile, filepath.Join(f.TagDir, f.Filename))
	return err
}

func (f *File) ClearTemp() error {
	err := os.Remove(f.Tempfile)
	return err
}

func (f *File) ImagePath(width int, height int) string {
	return filepath.Join(f.TagDir, ".cache",
		strconv.Itoa(width)+"x"+strconv.Itoa(height)+"-"+
			f.Filename)
}

func (f *File) GenerateImage(width int, height int, crop bool) error {
	fpath := filepath.Join(f.TagDir, f.Filename)

	s, err := imaging.Open(fpath)
	if err != nil {
		return err
	}

	if crop {
		im := imaging.Fill(s, width, height, imaging.Center,
			imaging.Lanczos)
		err = imaging.Save(im, f.ImagePath(width, height))
	} else {
		im := imaging.Resize(s, width, height, imaging.Lanczos)
		err = imaging.Save(im, f.ImagePath(width, height))
	}
	return err
}

//func (f *File) ResizeImage(width int, height width, crop bool) error {
//	fpath := filepath.Join(f.TagDir, f.Filename)
//
//	s, err := imaging.Open(fpath)
//	if err != nil {
//		return err
//	}
//
//	thumb := imaging.Fill(s, width, height, imaging.Center,
//		imaging.NearestNeighbor)
//	err = imaging.Save(thumb, f.ThumbnailPath())
//	return err
//}

//func (f *File) GenerateTag() error {
//        var tag = randomString(16)
//        err := f.SetTag(tag)
//        return err
//}

func (f *File) Purge() error {
	for _, l := range f.Links {
		if err := purge(l.Href); err != nil {
			return err
		}
	}

	return nil
}

// Return the full URL from the links struct. Useful in templates.
func (f *File) GetLink(s string) string {
	link := ""
	for _, l := range f.Links {
		// Search for the Rel value s
		if l.Rel == s {
			link = l.Href
		}
	}
	return link
}

// Return DateTime as a string. Useful in templates.
func (f *File) DateTimeString() string {
	if f.DateTime.IsZero() {
		return ""
	}

	return f.DateTime.Format("2006-01-02 15:04:05 UTC")
}

func purge(url string) error {
	timeout := time.Duration(2 * time.Second)
	client := &http.Client{
		Timeout: timeout,
	}

	// Invalidate the file
	req, err := http.NewRequest("PURGE", url, nil)
	if err != nil {
		return err
	}

	_, err = client.Do(req)
	if err != nil {
		return err
	}
	// Should probably log the URL and response code
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
