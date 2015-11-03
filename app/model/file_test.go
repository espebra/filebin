package model

import (
	"testing"
	"math/rand"
	"io/ioutil"
	"os"
	"path/filepath"
)

func TestSetFilename(t *testing.T) {
	var err error
	f := File {}
	err = f.SetFilename("foo")
	if err != nil {
		t.Fatal(err)
	}
	if f.Filename != "foo" {
		t.Fatal("a Sanitizing failed:", f.Filename)
	}

	err = f.SetFilename(" foo!\"#$%&()= ")
	if err != nil {
		t.Fatal(err)
	}
	if f.Filename != "_foo________=_" {
		t.Fatal("b Sanitizing failed:", f.Filename)
	}

	err = f.SetFilename("/foo/bar/baz")
	if err != nil {
		t.Fatal(err)
	}
	if f.Filename != "_foo_bar_baz" {
		t.Fatal("c Sanitizing failed:", f.Filename)
	}

	e := ExtendedFile {}
	err = e.SetFilename("foo")
	if err != nil {
		t.Fatal(err)
	}
	if e.Filename != "foo" {
		t.Fatal("a Sanitizing failed:", e.Filename)
	}

	// Reset
	e = ExtendedFile {}
	e.Checksum = "123456789012345678901234567890"
	err = e.SetFilename("")
	if err != nil {
		e.SetFilename(e.Checksum)
	} else {
		t.Fatal("Should not accept empty filename")
	}
	if e.Filename != e.Checksum {
		t.Fatal("c Sanitizing failed " + e.Filename + " should be the checksum:", e.Checksum)
	}

}

func TestSetTagID(t *testing.T) {
	var err error

	f := File {}
	err = f.SetTagID("s")
	if err == nil {
		t.Fatal("Invalid tag specified.")
	}

	err = f.SetTagID(" s ")
	if err == nil {
		t.Fatal("Invalid tag specified.")
	}

	err = f.SetTagID("/foo/bar")
	if err == nil {
		t.Fatal("Invalid tag specified.")
	}

	err = f.SetTagID("../foo")
	if err == nil {
		t.Fatal("Invalid tag specified.")
	}

	err = f.SetTagID("abcdefghijklmnop")
	if err != nil {
		t.Fatal(err)
	}

	err = f.SetTagID("")
	if err != nil {
		err = f.GenerateTagID()
		if err != nil {
			t.Fatal(err)
		}
	}
	if f.TagID == "" {
		t.Fatal("The tag should not be empty")
	}
}

func TestRandomString(t *testing.T) {
	rand.Seed(1)
	str := randomString(16)
	if str != "fpllngzieyoh43e0" {
		t.Fatal("Random string from known seed is not", str)
	}
}

func TestDetectMIME(t *testing.T) {
	var err error

	f := File {}
	f.TagDir = "testdata"

	f.Filename = "image.png"
	err = f.DetectMIME()
	if err != nil {
		t.Fatal(err)
	}
	if f.MIME != "image/png" {
		t.Fatal("Unable to detect mime type:", f.MIME)
	}

	f.Filename = "image.jpg"
	err = f.DetectMIME()
	if err != nil {
		t.Fatal(err)
	}
	if f.MIME != "image/jpeg" {
		t.Fatal("Unable to detect mime type:", f.MIME)
	}

	f.Filename = "image.gif"
	err = f.DetectMIME()
	if err != nil {
		t.Fatal(err)
	}
	if f.MIME != "image/gif" {
		t.Fatal("Unable to detect mime type:", f.MIME)
	}

	f.Filename = "unknownfile"
	err = f.DetectMIME()
	if err == nil {
		t.Fatal("File does not exist.")
	}
	if f.MIME != "image/gif" {
		t.Fatal("Unable to detect mime type:", f.MIME)
	}
}

func TestEnsureDirectoryExists(t *testing.T) {
	// Use TempDir to figure out the path to a valid directory
	dir, err := ioutil.TempDir(os.TempDir(), "prefix")
	if err != nil {
		t.Fatal(err)
	}
	
	f := File {}
	f.SetTagID("foofoofoo")
	f.TagDir = filepath.Join(dir, f.TagID)

	err = f.EnsureTagDirectoryExists()
	if err != nil {
		t.Fatal("This directory cannot be created:", err)
	}

	// Ensure that the directory is created
	err = f.EnsureTagDirectoryExists()
	if err != nil {
		t.Fatal("This directory wasn't created:", err)
	}

	os.Remove(f.TagDir)
	if err != nil {
		t.Fatal(err)
	}

	// Remove the directory to clean up
	err = os.Remove(dir)
	if err != nil {
		t.Fatal(err)
	}
}

func TestIsDir(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "prefix")
	defer os.Remove(dir)
	if err != nil {
		t.Fatal(err)
	}
	if isDir(dir) != true {
		t.Fatal("Unable to detect " + dir + " as a directory")
	}

	if isDir("/unknowndirectory") != false {
		t.Fatal("Non existing path should not be a directory")
	}

	file, err := ioutil.TempFile(os.TempDir(), "prefix")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())
	if isDir(file.Name()) != false {
		t.Fatal("File", file.Name(), "is not a directory")
	}
}

func TestWriteTempfile(t *testing.T) {
	// Use TempDir to figure out the path to a valid directory
	dir, err := ioutil.TempDir(os.TempDir(), "prefix")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(dir)

	from_file, err := ioutil.TempFile(os.TempDir(), "prefix")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(from_file.Name())
	from_file.WriteString("some content")
	from_file.Sync()
	from_file.Seek(0, 0)

	f := ExtendedFile {}
	f.SetTagID("foo")
	f.SetFilename("bar")
	f.TagDir = filepath.Join(dir, f.TagID)
	err = f.EnsureTagDirectoryExists()
	if err != nil {
		t.Fatal(err)
	}
	err = f.WriteTempfile(from_file, dir)
	if err != nil {
		t.Fatal(err)
	}
	if f.Bytes != 12 {
		t.Fatal("The amount of bytes was unexpected:", f.Bytes)
	}
}

func TestPublish(t *testing.T) {
	// Use TempDir to figure out the path to a valid directory
	dir, err := ioutil.TempDir(os.TempDir(), "prefix")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(dir)

	f := ExtendedFile {}
	f.SetTagID("foo")
	f.SetFilename("bar")
	f.TagDir = filepath.Join(dir, f.TagID)

	f.Tempfile = "testdata/image.png"

	err = f.Publish()
	if err != nil {
		t.Fatal(err)
	}

	/// XXX: Verify the result
}

func TestGenerateLinks(t *testing.T) {
	f := ExtendedFile {}
	f.SetFilename("foo")
	f.SetTagID("validtag")
	f.GenerateLinks("http://localhost:8080")

	if len(f.Links) != 2 {
		t.Fatal("Unexpected amount of links:", len(f.Links))
	}
}

//func TestVerifySHA256(t *testing.T) {
//	// Use TempDir to figure out the path to a valid directory
//	dir, err := ioutil.TempDir(os.TempDir(), "prefix")
//	if err != nil {
//		t.Fatal(err)
//	}
//	defer os.Remove(dir)
//
//	from_file, err := ioutil.TempFile(os.TempDir(), "prefix")
//	if err != nil {
//		t.Fatal(err)
//	}
//	defer os.Remove(from_file.Name())
//	from_file.WriteString("some content")
//	from_file.Sync()
//	from_file.Seek(0, 0)
//
//	f := ExtendedFile {}
//	f.SetTagID("foo")
//	f.SetFilename("bar")
//	f.TagDir = filepath.Join(dir, f.TagID)
//	err = f.EnsureTagDirectoryExists()
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	err = f.WriteTempfile(from_file, dir)
//	if err != nil {
//		t.Fatal(err)
//	}
//	err = f.VerifySHA256("290f493c44f5d63d06b374d0a5abd292fae38b92cab2fae5efefe1b0e9347f56")
//	if err != nil {
//		t.Fatal(err)
//	}
//}
