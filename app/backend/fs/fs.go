package fs

import (
	"archive/tar"
	"archive/zip"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/disintegration/imaging"
	"github.com/dustin/go-humanize"
	"github.com/rwcarlsen/goexif/exif"
)

type Backend struct {
	sync.RWMutex
	filedir    string
	tempdir    string
	baseurl    string
	expiration int64
	Bytes      int64 `json:"bytes"`
	files      map[string]File
	filters    []string
	Log        *log.Logger `json:"-"`
}

type Bin struct {
	Bin             string    `json:"bin"`
	Bytes           int64     `json:"bytes"`
	BytesReadable   string    `json:"-"`
	ExpiresAt       time.Time `json:"expires"`
	ExpiresReadable string    `json:"-"`
	Expired         bool      `json:"-"`
	UpdatedAt       time.Time `json:"updated"`
	UpdatedReadable string    `json:"-"`
	Files           []File    `json:"files,omitempty"`
	Album           bool      `json:"-"`
}

type File struct {
	Filename  string    `json:"filename"`
	Bin       string    `json:"bin"`
	Bytes     int64     `json:"bytes"`
	MIME      string    `json:"mime"`
	CreatedAt time.Time `json:"created"`
	Links     []Link    `json:"links"`

	// Image specific attributes
	DateTime  time.Time `json:"datetime,omitempty"`
	Longitude float64   `json:"longitude,omitempty"`
	Latitude  float64   `json:"latitude,omitempty"`
}

type Link struct {
	Rel  string `json:"rel"`
	Href string `json:"href"`
}

func InitBackend(baseurl string, filedir string, tempdir string, expiration int64, log *log.Logger, filters []string) (Backend, error) {
	be := Backend{}

	fi, err := os.Lstat(filedir)
	if err != nil {
		return be, err
	}

	if fi.IsDir() {
		// Filedir exists as a directory.
		be.filedir = filedir
	} else {
		// Path exists, but is not a directory.
		err = errors.New("The specified filedir is not a directory.")
		return be, err
	}

	be.Lock()
	be.Log = log
	be.baseurl = baseurl
	be.expiration = expiration
	be.tempdir = tempdir
	be.filters = filters
	err = be.getAllMetaData()
	be.Unlock()
	return be, err
}

func (be *Backend) Info() string {
	return "FS backend from " + be.filedir
}

func (be *Backend) getBins() ([]string, error) {
	var bins []string

	entries, err := ioutil.ReadDir(be.filedir)
	if err != nil {
		return bins, err
	}

	for _, entry := range entries {
		// Do not care about files
		if entry.IsDir() == false {
			continue
		}
		bins = append(bins, entry.Name())
	}

	return bins, nil
}

func (be *Backend) getFiles(bin string) ([]string, error) {
	var files []string

	path := filepath.Join(be.filedir, bin)
	entries, err := ioutil.ReadDir(path)
	if err != nil {
		return files, err
	}

	for _, entry := range entries {
		// Skip directories
		if entry.IsDir() == true {
			continue
		}

		// Skip files starting with .
		//if strings.HasPrefix(entry.Name(), ".") {
		//	continue
		//}
		files = append(files, entry.Name())
	}

	return files, nil
}

func (be *Backend) getAllMetaData() error {
	be.Log.Println("Reading all backend data")

	// Return metadata for all bins and files
	bins, err := be.getBins()
	if err != nil {
		return err
	}
	be.files = make(map[string]File)
	for _, bin := range bins {
		files, err := be.getFiles(bin)
		if err != nil {
			return err
		}
		for _, filename := range files {
			f, err := be.getFileMetaData(bin, filename)
			if err != nil {
				continue
			}
			id := f.Bin + f.Filename
			be.files[id] = f
		}
	}
	return nil
}

func (be *Backend) BinExists(bin string) bool {
	be.RLock()
	defer be.RUnlock()
	for _, f := range be.files {
		if f.Bin == bin {
			return true
		}
	}
	return false
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func (be *Backend) GetBins() []string {
	var bins []string
	be.RLock()
	defer be.RUnlock()
	for _, f := range be.files {
		if !stringInSlice(f.Bin, bins) {
			bins = append(bins, f.Bin)
		}
	}

	return bins
}

func (be *Backend) GetFiles(bin string) []string {
	var files []string
	be.RLock()
	defer be.RUnlock()
	for _, f := range be.files {
		if f.Bin != bin {
			continue
		}
		if !stringInSlice(f.Filename, files) {
			files = append(files, f.Filename)
		}
	}

	return files
}

func (be *Backend) GetBinsMetaData() []Bin {
	var bins []Bin

	for _, b := range be.GetBins() {
		bin, err := be.GetBinMetaData(b)
		if err != nil {
			// Log
		}

		bins = append(bins, bin)
	}
	sort.Sort(BinsByUpdatedAt(bins))

	return bins
}

func (be *Backend) NewBin(bin string) Bin {
	be.Log.Println("Generate new bin " + bin)

	b := Bin{}
	b.Bin = bin
	b.UpdatedAt = time.Now().UTC()
	b.ExpiresAt = b.UpdatedAt.Add(time.Duration(be.expiration) * time.Second)
	b.ExpiresReadable = humanize.Time(b.ExpiresAt)
	return b
}

func (be *Backend) GetBinMetaData(bin string) (Bin, error) {
	b := Bin{}
	be.RLock()
	for _, f := range be.files {
		if f.Bin != bin {
			continue
		}

		b.Bytes = b.Bytes + f.Bytes
		b.Files = append(b.Files, f)
		if strings.Split(f.MIME, "/")[0] == "image" {
			b.Album = true
		}

		if f.CreatedAt.After(b.UpdatedAt) {
			b.UpdatedAt = f.CreatedAt
		}
	}
	be.RUnlock()

	b.ExpiresAt = b.UpdatedAt.Add(time.Duration(be.expiration) * time.Second)

	b.BytesReadable = humanize.Bytes(uint64(b.Bytes))
	b.UpdatedReadable = humanize.Time(b.UpdatedAt)
	b.ExpiresReadable = humanize.Time(b.ExpiresAt)

	now := time.Now().UTC()
	if now.After(b.ExpiresAt) {
		b.Expired = true
	}

	sort.Sort(FilesByFilename(b.Files))
	sort.Sort(FilesByDateTime(b.Files))
	if len(b.Files) == 0 {
		err := errors.New("Bin " + bin + " does not exist")
		return b, err
	}
	b.Bin = bin
	return b, nil
}

func (be *Backend) DeleteBin(bin string) (Bin, error) {
	b, err := be.GetBinMetaData(bin)
	if err != nil {
		return b, err
	}

	bindir := filepath.Join(be.filedir, bin)
	be.Log.Println("Deleting bin " + bin + " (" + bindir + ")")

	if err := os.RemoveAll(bindir); err != nil {
		return b, err
	}

	be.Lock()
	defer be.Unlock()
	for id, f := range be.files {
		if f.Bin != bin {
			continue
		}
		delete(be.files, id)
	}
	return b, nil
}

func (be *Backend) GetBinArchive(bin string, format string, w http.ResponseWriter) (io.Writer, string, error) {
	be.Log.Println("Generating " + format + " archive of bin " + bin)

	var err error
	var paths []string

	be.RLock()
	for _, f := range be.files {
		if f.Bin != bin {
			continue
		}
		p := filepath.Join(be.filedir, bin, f.Filename)
		paths = append(paths, p)
	}
	be.RUnlock()

	var fp io.Writer
	if format == "zip" {
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", `attachment; filename="`+bin+`.zip"`)
		zw := zip.NewWriter(w)
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

			be.Log.Println("Added " + fname + " to the archive: " + strconv.FormatInt(bytes, 10) + " bytes")
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
			be.Log.Println("Added " + fname + " to the archive: " + strconv.FormatInt(bytes, 10) + " bytes")
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
	path := filepath.Join(be.filedir, bin, filename)
	fp, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	//defer fp.Close()
	return fp, err
}

func (be *Backend) GetThumbnail(bin string, filename string, width int, height int) (io.ReadSeeker, error) {
	cachedir := filepath.Join(be.filedir, bin, ".cache")
	path := filepath.Join(cachedir, strconv.Itoa(width)+"x"+strconv.Itoa(height)+"-"+filename)

	fp, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return fp, err
}

func (be *Backend) GetFileMetaData(bin string, filename string) (File, error) {
	f := File{}
	be.RLock()
	defer be.RUnlock()
	for _, f := range be.files {
		if f.Bin != bin {
			continue
		}
		if f.Filename != filename {
			continue
		}
		return f, nil
	}
	err := errors.New("File not found")
	return f, err
}

func (be *Backend) getFileMetaData(bin string, filename string) (File, error) {
	be.Log.Println("Reading file meta data: " + filename + " (" + bin + ")...")

	f := File{}
	path := filepath.Join(be.filedir, bin, filename)

	// File info
	fi, err := os.Lstat(path)
	if err != nil || fi.IsDir() == true {
		return f, errors.New("File does not exist.")
	}

	f.Bin = bin
	f.Filename = filename
	f.Bytes = fi.Size()
	f.CreatedAt = fi.ModTime()

	fp, err := os.Open(path)
	if err != nil {
		return f, err
	}
	defer fp.Close()

	// MIME
	buffer := make([]byte, 512)
	if _, err = fp.Seek(0, 0); err != nil {
		return f, err
	}
	if _, err = fp.Read(buffer); err != nil {
		return f, err
	}
	f.MIME = http.DetectContentType(buffer)
	f.Links = be.GenerateLinks(bin, filename)

	// Exif
	if strings.Split(f.MIME, "/")[0] == "image" {
		if _, err = fp.Seek(0, 0); err != nil {
			return f, err
		}

		exif, err := exif.Decode(fp)
		if err != nil {
			// XXX: Log
		} else {
			f.DateTime, err = exif.DateTime()
			if err != nil {
				// Could not read DateTime or DateTimeOriginal.
				// Certain Android phones (verified on Nexus 6) have a
				// bug where these fields may be missing from the EXIF
				// data in pictures. The capture time may be stored in
				// the GPS info, so let's try to assemble a valid
				// DateTime from it.
				dStamp, dErr := exif.Get("GPSDateStamp")
				tStamp, tErr := exif.Get("GPSTimeStamp")
				if dErr == nil && tErr == nil {
					s := dStamp.String() + " " + tStamp.String()
					s = strings.Replace(s, "\"", "", -1)
					s = strings.Replace(s, "[", "", -1)
					s = strings.Replace(s, "]", "", -1)
					s = strings.Replace(s, ":", "-", -1)
					s = strings.Replace(s, ",", " ", -1)
					s = strings.Replace(s, "/1", "", -1)
					be.Log.Println("String to parse: " + s)

					// After removing the special characters above, the
					// string to parse is on the following format:
					// 2016-05-21 12 26 12
					dt, err := time.Parse("2006-01-02 15 4 5", s)
					if err != nil {
						be.Log.Println("Unable to parse: ", err)
					} else {
						f.DateTime = dt.Local()
					}
				}
			}

			f.Latitude, f.Longitude, err = exif.LatLong()
			if err != nil {
				/// XXX: Log
			}
		}
	}

	return f, nil
}

func (be *Backend) GenerateThumbnail(bin string, filename string, width int, height int, crop bool) error {
	f, err := be.GetFileMetaData(bin, filename)
	if err != nil {
		return err
	}

	if strings.Split(f.MIME, "/")[0] != "image" {
		return errors.New("Batch job skipped: " + filename + " is not an image")
	}

	fpath := filepath.Join(be.filedir, bin, filename)
	cachedir := filepath.Join(be.filedir, bin, ".cache")
	if !isDir(cachedir) {
		if err := os.Mkdir(cachedir, 0700); err != nil {
			return err
		}
	}
	dst := filepath.Join(cachedir, strconv.Itoa(width)+"x"+strconv.Itoa(height)+"-"+filename)

	// Optimize to skip thumbnail generation if the thumbnail file exists
	// and is newer than the file.
	fi, err := os.Lstat(dst)
	if err == nil {
		if f.CreatedAt.After(fi.ModTime()) {
			// File newer than thumbnail. Ok to generate.
		} else {
			// File older than thumbnail. No need to generate.
			return nil
		}
	}

	s, err := imaging.Open(fpath)
	if err != nil {
		return err
	}

	if crop {
		im := imaging.Fill(s, width, height, imaging.Center, imaging.Lanczos)
		err = imaging.Save(im, dst)
	} else {
		im := imaging.Resize(s, width, height, imaging.Lanczos)
		err = imaging.Save(im, dst)
	}

	f.Links = be.GenerateLinks(f.Bin, f.Filename)
	be.Lock()
	defer be.Unlock()
	id := bin + filename
	delete(be.files, id)
	be.files[id] = f

	return err
}

func (be *Backend) UploadFile(bin string, filename string, data io.ReadCloser) (File, error) {
	be.Log.Println("Uploading file " + filename + " to bin " + bin)
	f := File{}
	f.Filename = filename
	f.Bin = bin

	if !isDir(be.tempdir) {
		if err := os.Mkdir(be.tempdir, 0700); err != nil {
			be.Log.Println("Unable to create directory: "+be.tempdir+":", err)
			err := errors.New("Unable to save file")
			return f, err
		}
	}

	fp, err := ioutil.TempFile(be.tempdir, "upload")
	// Defer removal of the tempfile to clean up partially uploaded files.
	defer os.Remove(fp.Name())
	defer fp.Close()

	if err != nil {
		be.Log.Println(err)
		err = errors.New("Unable to save file. Please try again later")
		return f, err
	}

	bytes, err := io.Copy(fp, data)
	if err != nil {
		be.Log.Println("Error occurred during io.Copy: " + err.Error())
		err = errors.New("Unable to save file. Please try again later")
		return f, err
	}
	be.Log.Println("Uploaded " + strconv.FormatInt(bytes, 10) + " bytes")

	f.Bytes = bytes
	if bytes == 0 {
		be.Log.Println("Empty files are not allowed. Aborting.")

		if err := os.Remove(fp.Name()); err != nil {
			be.Log.Println(err)
			// This should not happen
			err = errors.New("Internal Server Error")
			return f, err
		}

		err := errors.New("The file size must be more than 0 bytes")
		return f, err
	}

	buffer := make([]byte, 512)
	_, err = fp.Seek(0, 0)
	if err != nil {
		be.Log.Println(err)
		// This should not happen
		err = errors.New("Internal Server Error")
		return f, err
	}
	_, err = fp.Read(buffer)
	if err != nil {
		be.Log.Println(err)
		// This should not happen
		err = errors.New("Internal Server Error")
		return f, err
	}
	f.MIME = http.DetectContentType(buffer)
	be.Log.Println("Detected content-type: " + f.MIME)

	// Filter and reject unwanted content-types here
	for _, filter := range be.filters {
		if strings.Contains(f.MIME, filter) {
			be.Log.Println("Matched content-type filter: " + filter)
			if err := os.Remove(fp.Name()); err != nil {
				be.Log.Println(err)
				// This should not happen
				err = errors.New("Internal Server Error")
				return f, err
			}

			err := errors.New("Illegal content-type")
			return f, err
		}
	}

	// Exif
	if strings.Split(f.MIME, "/")[0] == "image" {
		if _, err = fp.Seek(0, 0); err != nil {
			be.Log.Println(err)
			// This should not happen
			err = errors.New("Internal Server Error")
			return f, err
		}
		exif, err := exif.Decode(fp)
		if err != nil {
			// XXX: Log
		} else {
			f.DateTime, err = exif.DateTime()
			if err != nil {
				// Could not read DateTime or DateTimeOriginal.
				// Certain Android phones (verified on Nexus 6) have a
				// bug where these fields may be missing from the EXIF
				// data in pictures. The capture time may be stored in
				// the GPS info, so let's try to assemble a valid
				// DateTime from it.
				dStamp, dErr := exif.Get("GPSDateStamp")
				tStamp, tErr := exif.Get("GPSTimeStamp")
				if dErr == nil && tErr == nil {
					s := dStamp.String() + " " + tStamp.String()
					s = strings.Replace(s, "\"", "", -1)
					s = strings.Replace(s, "[", "", -1)
					s = strings.Replace(s, "]", "", -1)
					s = strings.Replace(s, ":", "-", -1)
					s = strings.Replace(s, ",", " ", -1)
					s = strings.Replace(s, "/1", "", -1)
					be.Log.Println("String to parse: " + s)

					// After removing the special characters above, the
					// string to parse is on the following format:
					// 2016-05-21 12 26 12
					dt, err := time.Parse("2006-01-02 15 4 5", s)
					if err != nil {
						be.Log.Println("Unable to parse: ", err)
					} else {
						f.DateTime = dt.Local()
					}
				}
			}

			f.Latitude, f.Longitude, err = exif.LatLong()
			if err != nil {
				/// XXX: Log
			}
		}
	}

	// iOS devices provide only one filename even when uploading
	// multiple images. Providing some workaround for this below.
	// XXX: Refactoring needed. Can probably be done nicer.
	if strings.Split(f.MIME, "/")[0] == "image" && !f.DateTime.IsZero() {
		dt := f.DateTime.Format("060102-150405")
		var fname string

		// List of filenames to modify
		if filename == "image.jpeg" {
			fname = "img-" + dt + ".jpeg"
		}
		if filename == "image.gif" {
			fname = "img-" + dt + ".gif"
		}
		if filename == "image.png" {
			fname = "img-" + dt + ".png"
		}

		if fname != "" {
			be.Log.Println("Filename workaround triggered")
			be.Log.Println("Filename modified: " + fname)
			f.Filename = fname
		}
	}

	bindir := filepath.Join(be.filedir, bin)
	if !isDir(bindir) {
		if err := os.Mkdir(bindir, 0700); err != nil {
			be.Log.Println(err)
			// Possibly due to full disk
			err = errors.New("Unable to save file. Please try again later")
			return f, err
		}
	}

	dst := filepath.Join(bindir, f.Filename)
	be.Log.Println("Copying contents to " + dst)
	if err := CopyFile(fp.Name(), dst); err != nil {
		be.Log.Println(err)
		// Possibly due to full disk
		err = errors.New("Unable to save file. Please try again later")
		return f, err
	}

	fi, err := os.Lstat(dst)
	if err != nil {
		be.Log.Println(err)
		// This should not happen
		err = errors.New("Internal Server Error")
		return f, err
	}

	f.CreatedAt = fi.ModTime()
	f.Links = be.GenerateLinks(bin, f.Filename)

	be.Lock()
	defer be.Unlock()
	id := f.Bin + f.Filename
	delete(be.files, id)
	be.files[id] = f

	return f, err
}

func (be *Backend) DeleteFile(bin string, filename string) (File, error) {
	f, err := be.GetFileMetaData(bin, filename)
	if err != nil {
		return f, err
	}
	fpath := filepath.Join(be.filedir, bin, filename)
	if err := os.Remove(fpath); err != nil {
		return f, err
	}

	be.Lock()
	defer be.Unlock()
	id := bin + filename
	delete(be.files, id)
	return f, nil
}

func (be *Backend) GenerateLinks(bin string, filename string) []Link {
	links := []Link{}

	// Links
	fileLink := Link{}
	fileLink.Rel = "file"
	fileLink.Href = be.baseurl + "/" + bin + "/" + filename
	links = append(links, fileLink)

	binLink := Link{}
	binLink.Rel = "bin"
	binLink.Href = be.baseurl + "/" + bin
	links = append(links, binLink)

	cachedir := filepath.Join(be.filedir, bin, ".cache")
	if isFile(filepath.Join(cachedir, "115x115-"+filename)) {
		thumbLink := Link{}
		thumbLink.Rel = "thumbnail"
		thumbLink.Href = be.baseurl + "/" + bin + "/" + filename + "?width=115&height=115"
		links = append(links, thumbLink)
	}

	if isFile(filepath.Join(cachedir, "1140x0-"+filename)) {
		albumItemLink := Link{}
		albumItemLink.Rel = "album item"
		albumItemLink.Href = be.baseurl + "/" + bin + "/" + filename + "?width=1140"
		links = append(links, albumItemLink)

		albumLink := Link{}
		albumLink.Rel = "album"
		albumLink.Href = be.baseurl + "/album/" + bin
		links = append(links, albumLink)
	}
	return links
}

func (f File) BytesReadable() string {
	return humanize.Bytes(uint64(f.Bytes))
}

func (f *File) CreatedReadable() string {
	return humanize.Time(f.CreatedAt)
}

func (f *File) DateTimeReadable() string {
	return humanize.Time(f.DateTime)
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

// Sort files by filename
type FilesByFilename []File

func (a FilesByFilename) Len() int {
	return len(a)
}

func (a FilesByFilename) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a FilesByFilename) Less(i, j int) bool {
	return a[i].Filename < a[j].Filename
}

// Sort files by DateTime
type FilesByDateTime []File

func (a FilesByDateTime) Len() int {
	return len(a)
}

func (a FilesByDateTime) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a FilesByDateTime) Less(i, j int) bool {
	return a[i].DateTime.Before(a[j].DateTime)
}

// Sort bins by Update At
type BinsByUpdatedAt []Bin

func (a BinsByUpdatedAt) Len() int {
	return len(a)
}

func (a BinsByUpdatedAt) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a BinsByUpdatedAt) Less(i, j int) bool {
	return a[i].UpdatedAt.After(a[j].UpdatedAt)
}
