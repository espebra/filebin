package model

import (
	"github.com/GeertJohan/go.rice"
	"github.com/espebra/filebin/app/backend/fs"
	"github.com/espebra/filebin/app/config"
	"github.com/espebra/filebin/app/events"
	"github.com/espebra/filebin/app/metrics"
	"github.com/espebra/filebin/app/tokens"
	"log"
)

type Job struct {
	Bin      string
	Filename string
	Log      *log.Logger
	Cfg      *config.Configuration
}

type Context struct {
	TemplateBox *rice.Box
	StaticBox   *rice.Box
	Baseurl     string
	Log         *log.Logger
	WorkQueue   chan Job
	Backend     *fs.Backend
	Metrics     *metrics.Metrics
	Events      *events.Events
	Tokens      *tokens.Tokens
	Token       string
	RemoteAddr  string
}
