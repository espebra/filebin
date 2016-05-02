package fs

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"testing"
	"time"
)

const (
	CONTENT    = "Some content"
	EXPIRATION = 3
)

var (
	be Backend
)

func TestMain(m *testing.M) {
	log := log.New(os.Stdout, "- ", log.LstdFlags)

	baseurl := "http://127.0.0.1"

	filedir, err := ioutil.TempDir("", "filebin-filedir")
	if err != nil {
		log.Fatal(err)
	}

	tempdir, err := ioutil.TempDir("", "filebin-tempdir")
	if err != nil {
		log.Fatal(err)
	}

	be, err = InitBackend(baseurl, filedir, tempdir, int64(EXPIRATION), log)
	if err != nil {
		log.Println(err)
		os.Exit(2)
	}

	retCode := m.Run()

	// Clean up
	os.RemoveAll(filedir)
	os.RemoveAll(tempdir)

	os.Exit(retCode)
}

func TestInfo(t *testing.T) {
	info := be.Info()
	expected := "FS backend from " + be.filedir
	if info != expected {
		t.Fatal("Unexpected info string: " + info)
	}
}

func TestUploadFile(t *testing.T) {
	bin := "testbin"
	filename := "testfile"
	data := ioutil.NopCloser(bytes.NewBufferString(CONTENT))

	f, err := be.UploadFile(bin, filename, data)
	if err != nil {
		t.Fatal(err)
	}
	if f.Filename != "testfile" {
		t.Fatal("Unexpected filename: " + f.Filename)
	}
	if f.Bytes != 12 {
		t.Fatal("Unexpected file size: " + strconv.FormatInt(f.Bytes, 10))
	}
	if f.MIME != "application/octet-stream" {
		t.Fatal("Unexpected MIME: " + f.MIME)
	}
	numLinks := len(f.Links)
	if numLinks != 2 {
		t.Fatal("Unexpected number of links: " + strconv.Itoa(numLinks))
	}

	files, err := be.getFiles(bin)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatal("Unexpected number of files: " + strconv.Itoa(len(files)))
	}
}

func TestGetFileMetaData(t *testing.T) {
	bin := "testbin"
	filename := "testfile"

	f, err := be.GetFileMetaData(bin, filename)
	if err != nil {
		t.Fatal(err)
	}
	if f.Filename != "testfile" {
		t.Fatal("Unexpected filename: " + f.Filename)
	}
	if f.Bytes != 12 {
		t.Fatal("Unexpected file size: " + strconv.FormatInt(f.Bytes, 10))
	}
	if f.Checksum != "9c6609fc5111405ea3f5bb3d1f6b5a5efd19a0cec53d85893fd96d265439cd5b" {
		t.Fatal("Unexpected checksum: " + f.Checksum)
	}
	if f.MIME != "application/octet-stream" {
		t.Fatal("Unexpected MIME: " + f.MIME)
	}
}

func TestGetFile(t *testing.T) {
	bin := "testbin"
	filename := "testfile"

	fp, err := be.GetFile(bin, filename)
	if err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(fp)
	content := buf.String()
	if content != CONTENT {
		t.Fatal("Unexpected content: " + content)
	}
}

func TestGetBinMetaData(t *testing.T) {
	bin := "testbin"

	b, err := be.GetBinMetaData(bin)
	if err != nil {
		t.Fatal(err)
	}

	if b.Bytes != 12 {
		t.Fatal("Unexpected bin size: " + strconv.FormatInt(b.Bytes, 10))
	}

	if b.Bin != bin {
		t.Fatal("Unexpected bin id: " + b.Bin)
	}

	fileNum := len(b.Files)
	if fileNum != 1 {
		t.Fatal("Unexpected file count: " + strconv.Itoa(fileNum))
	}

	if b.Expired != false {
		t.Fatal("Bin has unexpectedly expired")
	}
}

func TestAllMetaData(t *testing.T) {
	fileNum := len(be.files)
	if fileNum != 1 {
		t.Fatal("Unexpected file count: " + strconv.Itoa(fileNum))
	}
}

func TestExpiredBin(t *testing.T) {
	time.Sleep(EXPIRATION * time.Second)

	bin := "testbin"
	b, err := be.GetBinMetaData(bin)
	if err != nil {
		t.Fatal(err)
	}

	if b.Expired != true {
		t.Fatal("Bin has unexpectedly not expired")
	}
}

func TestDeleteFile(t *testing.T) {
	bin := "testbin"
	filename := "testfile"

	if err := be.DeleteFile(bin, filename); err != nil {
		t.Fatal(err)
	}

	_, err := be.GetFileMetaData(bin, filename)
	if err == nil {
		t.Fatal("Unexpected success when reading deleted file")
	}

	err = be.DeleteFile(bin, filename)
	expected := "File " + filename + " does not exist in bin " + bin + "."
	if err.Error() != expected {
		t.Fatal(err)
	}
}

func TestDeleteBin(t *testing.T) {
	bin := "testbin"
	if err := be.DeleteBin(bin); err != nil {
		t.Fatal(err)
	}

	_, err := be.GetBinMetaData(bin)
	if err == nil {
		t.Fatal("Unexpected success when reading deleted bin")
	}

	err = be.DeleteBin(bin)
	expected := "Bin " + bin + " does not exist."
	if err.Error() != expected {
		t.Fatal(err)
	}
}
