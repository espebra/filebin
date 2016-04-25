package fs

import (
	"os"
	"fmt"
	"time"
	"errors"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/ioutil"
	"path/filepath"
	"net/http"
	"strings"
	//"sort"

	"github.com/dustin/go-humanize"
)

type Backend struct {
	filedir            string
	baseurl            string
	expiration         int64
        Bytes              int64           `json:"bytes"`
	Bins               []Bin
}

type Bin struct {
	Bin                string          `json:"bin"`
        Bytes              int64           `json:"bytes"`
        ExpiresAt          time.Time       `json:"expires"`
        Expired            bool            `json:"-"`
        UpdatedAt          time.Time       `json:"updated"`
	Files              []File          `json:"files"`
	Album bool `json:"-"`
}

type File struct {
	Filename        string    `json:"filename"`
	Bytes           int64     `json:"bytes"`
	MIME            string    `json:"mime"`
	CreatedAt       time.Time `json:"created"`
	Checksum          string    `json:"checksum,omitempty"`
	Algorithm          string    `json:"algorithm,omitempty"`
	Links           []link    `json:"links"`
	//Verified        bool      `json:"verified"`
	//RemoteAddr      string    `json:"-"`
	//UserAgent       string    `json:"-"`
	//Tempfile        string    `json:"-"`
	
	// Image specific attributes
	DateTime         time.Time  `json:"datetime,omitempty"`
	Longitude        float64    `json:"longitude,omitempty"`
	Latitude         float64    `json:"latitude,omitempty"`
	Altitude         string     `json:"altitude,omitempty"`
	//Exif             *exif.Exif `json:"-"`
}

type link struct {
	Rel	string    `json:"rel"`
	Href	string    `json:"href"`
}

func InitBackend(baseurl string, filedir string, expiration int64) (Backend, error) {
	be := Backend{}

        fi, err := os.Lstat(filedir)
        if err == nil {
	        if fi.IsDir() {
			// Filedir exists as a directory.
			be.filedir = filedir
	        } else {
			// Path exists, but is not a directory.
	                err = errors.New("The specified filedir is not a directory.")
	        }
	}

	be.baseurl = baseurl
	be.expiration = expiration
	return be, err
}

func (be *Backend) Info() string {
	return "FS backend from " + be.filedir
}

func (be *Backend) GetAllMetaData() (*Backend, error) {
	// Return metadata for all bins and files
	path := be.filedir
	bins, err := ioutil.ReadDir(path)
	if err != nil {
		return be, err
	}

	for _, bin := range bins {
		// Do not care about files
		if bin.IsDir() == false {
			continue
		}
	
		b, err := be.GetBinMetaData(bin.Name())
		if err != nil {
			continue
		}
		be.Bytes = be.Bytes + b.Bytes
		be.Bins = append(be.Bins, b)
	}

	return be, nil
}

func (be *Backend) GetBinMetaData(bin string) (Bin, error) {
	fmt.Println("Bin meta data: " + bin)

	b := Bin{}
	path := filepath.Join(be.filedir, bin)

	// Directory info
	fi, err := os.Lstat(path)
	if err != nil  {
		return b, err
	}
	if fi.IsDir() == false {
		return b, errors.New("Bin does not exist.")
	}

	b.UpdatedAt = fi.ModTime()
	b.ExpiresAt = b.UpdatedAt.Add(time.Duration(be.expiration) * time.Second)
	b.Bytes = 0
	b.Bin = bin

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return b, err
	}

	for _, file := range files {
		// Do not care about sub directories (such as .cache)
		if file.IsDir() == true {
			continue
		}
	
		f, err := be.GetFileMetaData(bin, file.Name())
		if err != nil {
			continue
		}
		b.Bytes = b.Bytes + f.Bytes
		b.Files = append(b.Files, f)
	}

	return b, err
}

func (be *Backend) GetFile(bin string, filename string) (io.ReadSeeker, error) {
	fmt.Println("File contents: " + filename)

	path := filepath.Join(be.filedir, bin, filename)
	fp, err := os.Open(path)
	if err != nil {
		return fp, err
	}
	//defer fp.Close()
	return fp, err
}

func (be *Backend) GetFileMetaData(bin string, filename string) (File, error) {
	fmt.Println("File meta data: " + filename)

	f := File{}
	path := filepath.Join(be.filedir, bin, filename)

	// File info
	fi, err := os.Lstat(path)
	if err != nil || fi.IsDir() == true {
		return f, errors.New("File does not exist.")
	}

	f.Filename = filename
	f.Bytes = fi.Size()
	f.CreatedAt = fi.ModTime()

	// Calculate checksum
        fp, err := os.Open(path)
        if err != nil {
                return f, err
        }
        defer fp.Close()

        hash := sha256.New()
        _, err = io.Copy(hash, fp)
        if err != nil {
                return f, err
        }
	var result []byte
	f.Checksum = hex.EncodeToString(hash.Sum(result))
	f.Algorithm = "sha256"

	// MIME
        buffer := make([]byte, 512)
        _, err = fp.Seek(0, 0)
        if err != nil {
                return f, err
        }
        _, err = fp.Read(buffer)
        if err != nil {
                return f, err
        }
        f.MIME = http.DetectContentType(buffer)

	// Links
	fileLink := link{}
	fileLink.Rel = "file"
	fileLink.Href = be.baseurl + "/" + bin + "/" + filename
	f.Links = append(f.Links, fileLink)

	binLink := link{}
	binLink.Rel = "bin"
	binLink.Href = be.baseurl + "/" + bin
	f.Links = append(f.Links, binLink)

	return f, nil
}

func (be *Backend) UploadFile(bin string, filename string, data io.ReadCloser) (File, error) {
	var err error
	f := File{}
	f.Filename = filename

	//path := filepath.Join(be.filedir, bin, filename)
	//fp, err := os.Open(path)
	//if err != nil {
	//	return fp, err
	//}
	//defer fp.Close()
	return f, err
}

func (b Bin) BytesReadable() string {
        return humanize.Bytes(uint64(b.Bytes))
}

func (b Bin) UpdatedReadable() string {
        return humanize.Time(b.UpdatedAt)
}

func (b Bin) ExpiresReadable() string {
        return humanize.Time(b.ExpiresAt)
}

func (f File) BytesReadable() string {
        return humanize.Bytes(uint64(f.Bytes))
}

func (f *File) CreatedReadable() string {
        return humanize.Time(f.CreatedAt)
}

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

func (f *File) MediaType() string {
        s := strings.Split(f.MIME, "/")
        if len(s) > 0 {
                return s[0]
        }
        return ""
}

func (f *File) DateTimeString() string {
        if f.DateTime.IsZero() {
                return ""
        }

        return f.DateTime.Format("2006-01-02 15:04:05")
}
