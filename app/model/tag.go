package model

import (
	"errors"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"time"
	"os"

	"github.com/dustin/go-humanize"
)

type Tag struct {
	Tag		    	string		`json:"tag"`
	TagDir			string		`json:"-"`
	ExpirationAt		time.Time	`json:"-"`
	ExpirationReadable	string		`json:"expiration"`
	Expired			bool		`json:"-"`
	LastUpdateAt		time.Time	`json:"-"`
	LastUpdateReadable	string		`json:"lastupdate"`
	Files			[]interface{}	`json:"files"`
}


//func (t *Tag) GenerateTag() error {
//	var tag = randomString(16)
//	err := t.SetTag(tag)
//	return err
//}

func (t *Tag) SetTag(s string) error {
	validTag := regexp.MustCompile("^[a-zA-Z0-9-_]{8,}$")
	if validTag.MatchString(s) == false {
		return errors.New("Invalid tag specified. It contains " +
			"illegal characters or is too short")
	}
	t.Tag = s
	return nil
}

func (t *Tag) SetTagDir(filedir string) {
	t.TagDir = filepath.Join(filedir, t.Tag)
}

func (t *Tag) TagDirExists() bool {
	if isDir(t.TagDir) {
		return true
	} else {
		return false
	}
}

func (t *Tag) Info() error {
	if isDir(t.TagDir) == false {
		return errors.New("Tag does not exist.")
	}
	
	i, err := os.Lstat(t.TagDir)
	if err != nil {
		return err
	}
	t.LastUpdateAt = i.ModTime().UTC()
	t.LastUpdateReadable = humanize.Time(t.LastUpdateAt)
	return nil
}

func (t *Tag) IsExpired(expiration int64) (bool, error) {
        now := time.Now().UTC()

        // Calculate if the tag is expired or not
        if now.Before(t.ExpirationAt) {
                // Tag still valid
		return false, nil
        } else {
                // Tag expired
                t.Expired = true
		return true, nil
        }
}

func (t *Tag) CalculateExpiration(expiration int64) error {
	i, err := os.Lstat(t.TagDir)
	if err == nil {
		t.ExpirationAt = i.ModTime().UTC().Add(time.Duration(expiration) * time.Second)
	} else {
		t.ExpirationAt = time.Now().UTC().Add(time.Duration(expiration) * time.Second)
	}
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
		err = f.Info()
		if err != nil {
			return err
		}
		err = f.DetectMIME ()
		if err != nil {
			return err
		}

		//f.ExpirationAt = t.ExpirationAt
		//f.ExpirationReadable = t.ExpirationReadable

		f.GenerateLinks(baseurl)
		t.Files = append(t.Files, f)
	}
	return err
}
