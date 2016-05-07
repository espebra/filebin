package model

import (
	"github.com/GeertJohan/go.rice"
	"github.com/espebra/filebin/app/backend/fs"
	"github.com/espebra/filebin/app/stats"
	"log"
)

type Job struct {
	Bin      string
	Filename string
}

type Context struct {
	TemplateBox *rice.Box
	StaticBox   *rice.Box
	Baseurl     string
	Log         *log.Logger
	WorkQueue   chan Job
	Backend     *fs.Backend
	Stats       *stats.Stats
}
