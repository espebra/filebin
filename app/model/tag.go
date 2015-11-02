package model

import (
	"errors"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"time"
	"os"
	//"github.com/golang/glog"
	"github.com/dustin/go-humanize"
)

type Tag struct {
	Tag		    	string		`json:"tag"`
	TagDir			string		`json:"-"`
	LastUpdateReadable	string		`json:"lastupdate"`
	ExpirationReadable	string		`json:"expiration"`
	LastUpdateAt		time.Time	`json:"lastupdate_rfc3339"`
	ExpirationAt		time.Time	`json:"expiration_rfc3339"`
	Files			[]File		`json:"files"`
	//Links			[]Link		`json:"links"`
}

func (t *Tag) SetTag(s string, filedir string) error {
	validTag := regexp.MustCompile("^[a-zA-Z0-9-_]{8,}$")
	if validTag.MatchString(s) == false {
		return errors.New("Invalid tag specified. It contains " +
			"illegal characters or is too short")
	}
	t.Tag = s
	t.TagDir = filepath.Join(filedir, s)
	return nil
}

func (t *Tag) Exists() bool {
	if isDir(t.TagDir) {
		return true
	} else {
		return false
	}
}

func (t *Tag) Info(expiration int) error {
	if isDir(t.TagDir) == false {
		return errors.New("Tag does not exist.")
	}
	
	i, err := os.Lstat(t.TagDir)
	if err != nil {
		return err
	}
	t.LastUpdateAt = i.ModTime().UTC()
	t.LastUpdateReadable = humanize.Time(t.LastUpdateAt)
	t.ExpirationAt = i.ModTime().UTC().Add(time.Duration(expiration) * time.Second)
	t.ExpirationReadable = humanize.Time(t.ExpirationAt)
	return nil
}

func (t *Tag) List(baseurl string) error {
	var err error
	files, err := ioutil.ReadDir(t.TagDir)
	for _, file := range files {
		var f = File {}
		f.SetFilename(file.Name())
		f.SetTag(t.Tag)
		f.TagDir = t.TagDir
		err = f.Info(0)
		if err != nil {
			return err
		}
		err = f.DetectMIME ()
		if err != nil {
			return err
		}

		f.GenerateLinks(baseurl)
		t.Files = append(t.Files, f)
	}
	return err
}
