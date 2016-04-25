package model

import (
	"github.com/GeertJohan/go.rice"
	"log"
        "github.com/espebra/filebin/app/backend/fs"
)

type Context struct {
	TemplateBox *rice.Box
	StaticBox   *rice.Box
	Baseurl     string
	Log         *log.Logger
	WorkQueue   chan File
	Backend     *fs.Backend
}
