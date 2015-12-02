package model

import (
	"time"
)

type Image struct {
	DateTime		time.Time	`json:"datetime"`
	Longitude		string		`json:"longitude"`
	Latitude		string		`json:"latitude"`
	Altitude		string		`json:"altitude"`
	Thumbnail		bool
}

