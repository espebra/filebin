package model

import (
	"log"
	"github.com/GeertJohan/go.rice"
)

type Context struct {
	TemplateBox	*rice.Box
	StaticBox	*rice.Box
	Baseurl		string
	Log		*log.Logger
	WorkQueue	chan File
}
