package model

import (
        "github.com/GeertJohan/go.rice"
        "log"
)

type Context struct {
	TemplateBox	*rice.Box
	StaticBox	*rice.Box
	Baseurl		string
	Log		*log.Logger
}
