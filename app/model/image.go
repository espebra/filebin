package model

import (
	"time"
	"os"

        "github.com/rwcarlsen/goexif/exif"
)

type Image struct {
	DateTime		time.Time	`json:"datetime"`
	Longitude		float64		`json:"longitude"`
	Latitude		float64		`json:"latitude"`
	Altitude		string		`json:"altitude"`
	Thumbnail		bool		`json:"thumbnail"`
	Exif			*exif.Exif	`json:"-"`
}

func (i *Image) ExtractExif(fpath string) error {
        fp, err := os.Open(fpath)
        if err != nil {
                return err
        }
        defer fp.Close()

        i.Exif, err = exif.Decode(fp)
        if err != nil {
                return err
        }

        //fmt.Println(i.Exif.String())
        return err
}

func (i *Image) ParseExif() error {
	var err error
	i.DateTime, err = i.Exif.DateTime()
	if err != nil {
        	return err
	}

	i.Latitude, i.Longitude, err = i.Exif.LatLong()
	if err != nil {
        	return err
	}
        return err
}
