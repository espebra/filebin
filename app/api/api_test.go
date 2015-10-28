package api

import (
	"testing"
	"math/rand"
	"io/ioutil"
	"os"
)

func TestTriggers(t *testing.T) {
	var err error

	err = triggerNewTagHandler("/bin/echo", "tag")
	if err != nil {
		t.Fatal(err)
	}

	err = triggerUploadedFileHandler("/bin/echo", "tag", "filename")
	if err != nil {
		t.Fatal(err)
	}

	err = triggerExpiredTagHandler("/bin/echo", "tag")
	if err != nil {
		t.Fatal(err)
	}

	err = triggerNewTagHandler("unknowncommand", "tag")
	if err == nil {
		t.Fatal("This should fail")
	}

	err = triggerUploadedFileHandler("unknowncommand", "tag", "filename")
	if err == nil {
		t.Fatal("This should fail")
	}

	err = triggerExpiredTagHandler("unknowncommand", "tag")
	if err == nil {
		t.Fatal("This should fail")
	}
}

func TestSha256Sum(t *testing.T) {
	file, err := ioutil.TempFile(os.TempDir(), "prefix")
	if err != nil {
		t.Fatal(err)
	}

	defer os.Remove(file.Name())
	file.WriteString("some content")
	file.Sync()
	checksum, err := sha256sum(file.Name())
	if err != nil {
		t.Fatal(err)
	}

	if checksum != "290f493c44f5d63d06b374d0a5abd292fae38b92cab2fae5efefe1b0e9347f56" {
		t.Fatal("Invalid checksum", checksum)
	}

	checksum, err = sha256sum("unknownfile")
	if err == nil {
		t.Fatal("This should fail")
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

func TestRandomString(t *testing.T) {
	rand.Seed(1)
	str := randomString(16)
	if str != "fpllngzieyoh43e0" {
		t.Fatal("Random string from known seed is not", str)
	}
}

func TestSanitizeFilename(t *testing.T) {
	var str string
	str = sanitizeFilename("foo")
	if str != "foo" {
		t.Fatal("Sanitizing failed:", str)
	}

	str = sanitizeFilename(" foo!\"#$%&()= ")
	if str != "foo________=" {
		t.Fatal("Sanitizing failed:", str)
	}

	str = sanitizeFilename("/foo/bar/baz")
	if str != "baz" {
		t.Fatal("Sanitizing failed:", str)
	}
}

func TestValidTag(t *testing.T) {
	if validTag("s") == true {
		t.Fatal("Too short tag")
	}

	if validTag("s ") == true {
		t.Fatal("Tag contains whitespace")
	}

	if validTag("/foo/bar") == true {
		t.Fatal("Tag contains invalid characters")
	}

	if validTag("../foo") == true {
		t.Fatal("Tag contains invalid characters")
	}

	if validTag("abcdefghijklmnop") == false {
		t.Fatal("This tag is valid")
	}
}

func TestEnsureDirectoryExists(t *testing.T) {
	// Use TempDir to figure out the path to a valid directory
	dir, err := ioutil.TempDir(os.TempDir(), "prefix")
	if err != nil {
		t.Fatal(err)
	}
	// Remove the directory and keep the path
	err = os.Remove(dir)
	if err != nil {
		t.Fatal(err)
	}
	
	// Try to create the directory using our function
	err = ensureDirectoryExists(dir)
	if err != nil {
		t.Fatal("This directory cannot be created:", err)
	}

	// Ensure that the directory is created
	err = ensureDirectoryExists(dir)
	if err != nil {
		t.Fatal("This directory wasn't created:", err)
	}

	// Remove the directory to clean up
	err = os.Remove(dir)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWriteToFile(t *testing.T) {
	from_file, err := ioutil.TempFile(os.TempDir(), "prefix")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(from_file.Name())
	from_file.WriteString("some content")
	from_file.Sync()
	from_file.Seek(0, 0)

	to_file, err := ioutil.TempFile(os.TempDir(), "prefix")
	if err != nil {
		t.Fatal(err)
	}
	os.Remove(to_file.Name())

	nBytes, err := writeToFile(to_file.Name(), from_file)
	if err != nil {
		t.Fatal(err)
	}
	if nBytes != 12 {
		t.Fatal("The amount of bytes was unexpected:", nBytes)
	}
}

func TestDetectMIME(t *testing.T) {
	var mime string
	var err error

	mime, err = detectMIME("testing/image.png")
	if err != nil {
		t.Fatal(err)
	}
	if mime != "image/png" {
		t.Fatal("Unable to detect mime type:", mime)
	}

	mime, err = detectMIME("testing/image.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if mime != "image/jpeg" {
		t.Fatal("Unable to detect mime type:", mime)
	}

	mime, err = detectMIME("testing/image.gif")
	if err != nil {
		t.Fatal(err)
	}
	if mime != "image/gif" {
		t.Fatal("Unable to detect mime type:", mime)
	}
}
