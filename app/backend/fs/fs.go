package fs

import (
	"archive/tar"
	"archive/zip"
	"compress/flate"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	//"sort"
	"log"

	"github.com/dustin/go-humanize"
)

type Backend struct {
	filedir    string
	tempdir    string
	baseurl    string
	expiration int64
	Bytes      int64 `json:"bytes"`
	Bins       []Bin
	Log        *log.Logger `json:"-"`
}

type Bin struct {
	Bin       string    `json:"bin"`
	Bytes     int64     `json:"bytes"`
	ExpiresAt time.Time `json:"expires"`
	UpdatedAt time.Time `json:"updated"`
	Files     []File    `json:"files"`
	Album     bool      `json:"-"`
}

type File struct {
	Filename  string    `json:"filename"`
	Bytes     int64     `json:"bytes"`
	MIME      string    `json:"mime"`
	CreatedAt time.Time `json:"created"`
	Checksum  string    `json:"checksum,omitempty"`
	Algorithm string    `json:"algorithm,omitempty"`
	Links     []link    `json:"links"`
	//Verified        bool      `json:"verified"`
	//RemoteAddr      string    `json:"-"`
	//UserAgent       string    `json:"-"`
	//Tempfile        string    `json:"-"`

	// Image specific attributes
	DateTime  *time.Time `json:"datetime,omitempty"`
	Longitude float64    `json:"longitude,omitempty"`
	Latitude  float64    `json:"latitude,omitempty"`
	Altitude  string     `json:"altitude,omitempty"`
	//Exif             *exif.Exif `json:"-"`
}

type link struct {
	Rel  string `json:"rel"`
	Href string `json:"href"`
}

func InitBackend(baseurl string, filedir string, tempdir string, expiration int64, log *log.Logger) (Backend, error) {
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

	be.Log = log
	be.baseurl = baseurl
	be.expiration = expiration
	be.tempdir = tempdir
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
	be.Log.Println("Reading bin meta data: " + bin)

	b := Bin{}
	path := filepath.Join(be.filedir, bin)

	// Directory info
	fi, err := os.Lstat(path)
	if err != nil {
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

func (be *Backend) GetBinArchive(bin string, format string, w http.ResponseWriter) (io.Writer, string, error) {
	be.Log.Println("Generate bin archive: " + bin)

	var err error

	//b := Bin{}
	path := filepath.Join(be.filedir, bin)

	// Directory info
	//fi, err := os.Lstat(path)
	//if err != nil  {
	//	return b, err
	//}
	//if fi.IsDir() == false {
	//	return b, errors.New("Bin does not exist.")
	//}

	//b.UpdatedAt = fi.ModTime()
	//b.ExpiresAt = b.UpdatedAt.Add(time.Duration(be.expiration) * time.Second)
	//b.Bytes = 0
	//b.Bin = bin

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, "", err
	}

	var paths []string

	for _, file := range files {
		// Do not care about sub directories (such as .cache)
		if file.IsDir() == true {
			continue
		}

		p := filepath.Join(path, file.Name())
		paths = append(paths, p)
	}

	var fp io.Writer
	if format == "zip" {
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", `attachment; filename="`+bin+`.zip"`)
		zw := zip.NewWriter(w)
		zw.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) {
			return flate.NewWriter(out, flate.BestSpeed)
		})

		for _, path := range paths {
			// Extract the filename from the absolute path
			fname := filepath.Base(path)
			//be.Log.Println("Adding to zip archive: " + fname)

			// Get stat info for modtime etc
			info, err := os.Stat(path)
			if err != nil {
				be.Log.Println(err)
				return nil, "", err
			}

			// Generate the Zip info header for this file based on the stat info
			header, err := zip.FileInfoHeader(info)
			if err != nil {
				be.Log.Println(err)
				return nil, "", err
			}

			ze, err := zw.CreateHeader(header)
			if err != nil {
				be.Log.Println(err)
				return nil, "", err
			}

			file, err := os.Open(path)
			if err != nil {
				be.Log.Println(err)
				return nil, "", err
			}

			bytes, err := io.Copy(ze, file)
			if err != nil {
				be.Log.Println(err)
				return nil, "", err
			}

			if err := file.Close(); err != nil {
				be.Log.Println(err)
				return nil, "", err
			}

			be.Log.Println("Added " + strconv.FormatInt(bytes, 10) + " bytes to the archive: " + fname)
		}
		if err := zw.Close(); err != nil {
			be.Log.Println(err)
			return nil, "", err
		}
		be.Log.Println("Zip archive generated successfully")
	} else if format == "tar" {
		w.Header().Set("Content-Type", "application/x-tar")
		w.Header().Set("Content-Disposition", `attachment; filename="`+bin+`.tar"`)
		tw := tar.NewWriter(w)
		for _, path := range paths {
			// Extract the filename from the absolute path
			fname := filepath.Base(path)
			//be.Log.Println("Adding to tar archive: " + fname)

			// Get stat info for modtime etc
			info, err := os.Stat(path)
			if err != nil {
				be.Log.Println(err)
				return nil, "", err
			}

			// Generate the tar info header for this file based on the stat info
			header, err := tar.FileInfoHeader(info, info.Name())
			if err != nil {
				be.Log.Println(err)
				return nil, "", err
			}

			if err := tw.WriteHeader(header); err != nil {
				be.Log.Println(err)
				return nil, "", err
			}

			file, err := os.Open(path)
			if err != nil {
				be.Log.Println(err)
				return nil, "", err
			}
			defer file.Close()
			bytes, err := io.Copy(tw, file)
			if err != nil {
				be.Log.Println(err)
				return nil, "", err
			}
			be.Log.Println("Added " + strconv.FormatInt(bytes, 10) + " bytes to the archive: " + fname)
		}
		if err := tw.Close(); err != nil {
			be.Log.Println(err)
			return nil, "", err
		}
		be.Log.Println("Tar archive generated successfully")
	} else {
		err = errors.New("Unsupported format")
	}

	archiveName := bin + "." + format

	return fp, archiveName, err
}

func (be *Backend) GetFile(bin string, filename string) (io.ReadSeeker, error) {
	be.Log.Println("File contents: " + bin + "/" + filename)

	path := filepath.Join(be.filedir, bin, filename)
	fp, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	//defer fp.Close()
	return fp, err
}

func (be *Backend) GetFileMetaData(bin string, filename string) (File, error) {
	be.Log.Println("File meta data: " + filename)

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
	f.Links = generateLinks(be.baseurl, bin, filename)

	return f, nil
}

func (be *Backend) UploadFile(bin string, filename string, data io.ReadCloser) (File, error) {
	be.Log.Println("Uploading file " + filename + " to bin " + bin)
	f := File{}

	if !isDir(be.tempdir) {
		if err := os.Mkdir(be.tempdir, 0700); err != nil {
			return f, err
		}
	}

	fp, err := ioutil.TempFile(be.tempdir, "upload")
	defer fp.Close()
	if err != nil {
		be.Log.Println(err)
		return f, err
	}

	bytes, err := io.Copy(fp, data)
	if err != nil {
		be.Log.Println(err)
		return f, err
	}
	be.Log.Println("Uploaded " + strconv.FormatInt(bytes, 10) + " bytes")

	if bytes == 0 {
		be.Log.Println("Empty files are not allowed. Aborting.")

		if err := os.Remove(fp.Name()); err != nil {
			be.Log.Println(err)
			return f, err
		}

		err := errors.New("No content. The file size must be more than 0 bytes")
		return f, err
	}

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

	bindir := filepath.Join(be.filedir, bin)
	if !isDir(bindir) {
		if err := os.Mkdir(bindir, 0700); err != nil {
			return f, err
		}
	}

	dst := filepath.Join(bindir, filename)
	be.Log.Println("Copying contents to " + dst)
	if err := CopyFile(fp.Name(), dst); err != nil {
		be.Log.Println(err)
		return f, err
	}

	be.Log.Println("Removing " + fp.Name())
	if err := os.Remove(fp.Name()); err != nil {
		be.Log.Println(err)
		return f, err
	}

	f.Filename = filename
	f.Bytes = bytes

	fi, err := os.Lstat(dst)
	if err != nil {
		be.Log.Println(err)
		return f, err
	}

	f.CreatedAt = fi.ModTime()
	f.Links = generateLinks(be.baseurl, bin, filename)

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

func (b Bin) Expired() bool {
	now := time.Now().UTC()
	if now.Before(b.ExpiresAt) {
		return false
	} else {
		return true
	}
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

func generateLinks(baseurl string, bin string, filename string) []link {
	links := []link{}

	// Links
	fileLink := link{}
	fileLink.Rel = "file"
	fileLink.Href = baseurl + "/" + bin + "/" + filename
	links = append(links, fileLink)

	binLink := link{}
	binLink.Rel = "bin"
	binLink.Href = baseurl + "/" + bin
	links = append(links, binLink)
	return links
}
