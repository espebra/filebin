package model

import (
	"time"
)

type Image struct {
	File
	DateTime		time.Time	`json:"datetime"`
	Thumbnail		bool
	GPS
}

