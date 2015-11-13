package model

import (
        "github.com/GeertJohan/go.rice"
)

type Context struct {
	TemplateBox	*rice.Box
	StaticBox	*rice.Box
}
