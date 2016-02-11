package api

import (
	"math/rand"
	"testing"
)

func TestTriggers(t *testing.T) {
	var err error

	err = triggerNewBinHandler("/bin/echo", "bin")
	if err != nil {
		t.Fatal(err)
	}

	err = triggerDeleteBinHandler("/bin/echo", "bin")
	if err != nil {
		t.Fatal(err)
	}

	err = triggerUploadFileHandler("/bin/echo", "bin", "filename")
	if err != nil {
		t.Fatal(err)
	}

	err = triggerDeleteFileHandler("/bin/echo", "bin", "filename")
	if err != nil {
		t.Fatal(err)
	}

	err = triggerNewBinHandler("unknowncommand", "bin")
	if err == nil {
		t.Fatal("This should fail")
	}

	err = triggerDeleteBinHandler("unknowncommand", "bin")
	if err == nil {
		t.Fatal("This should fail")
	}

	err = triggerUploadFileHandler("unknowncommand", "bin", "filename")
	if err == nil {
		t.Fatal("This should fail")
	}

	err = triggerDeleteFileHandler("unknowncommand", "bin", "filename")
	if err == nil {
		t.Fatal("This should fail")
	}
}

func TestRandomString(t *testing.T) {
	rand.Seed(1)
	str := randomString(16)
	if str != "fpllngzieyoh43e0" {
		t.Fatal("Random string from known seed is not", str)
	}
}
