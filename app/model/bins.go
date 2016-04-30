package model

import (
	"io/ioutil"
)

type Bins struct {
	Filedir    string
	Expiration int64
	Baseurl    string
	Bins       []Bin `json:"bins"`
}

func (l *Bins) Scan() error {

	files, err := ioutil.ReadDir(l.Filedir)
	if err != nil {
		return err
	}

	for _, f := range files {
		b := Bin{}
		if err := b.SetBin(f.Name()); err != nil {
			// Not a valid bin
			continue
		}

		b.SetBinDir(l.Filedir)
		b.CalculateExpiration(l.Expiration)

		if err := b.StatInfo(); err != nil {
			return err
		}

		if err := b.List(l.Baseurl); err != nil {
			return err
		}

		l.Bins = append(l.Bins, b)
	}

	return nil
}
