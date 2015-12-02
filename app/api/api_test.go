package api

import (
	"testing"
	"math/rand"
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

func TestRandomString(t *testing.T) {
      rand.Seed(1)
      str := randomString(16)
      if str != "fpllngzieyoh43e0" {
              t.Fatal("Random string from known seed is not", str)
      }
}
