package output

import (
	"io"
	//"io/ioutil"
	"net/http"
	"encoding/json"
	"strconv"
	"html/template"
	//"path/filepath"

        "github.com/golang/glog"
	//"github.com/GeertJohan/go.rice"

        "github.com/espebra/filebin/app/model"
)

func JSONresponse(w http.ResponseWriter, status int, h map[string]string, d interface{}) {
        dj, err := json.MarshalIndent(d, "", "    ")
        if err != nil {
                glog.Info("Unable to convert response to json: ", err)
                http.Error(w, "Failed while generating a response", http.StatusInternalServerError)
                return
        }

        for header, value := range h {
                w.Header().Set(header, value)
        }

        w.WriteHeader(status)
        //log.Info("Status " + strconv.Itoa(status))
        io.WriteString(w, string(dj))
        //Info.Print("Output: ", string(dj))
        glog.Info("Response status: " + strconv.Itoa(status))
}

// This function is a hack. Need to figure out a better way to do this.
func HTMLresponse(w http.ResponseWriter, tpl string, status int, h map[string]string, d interface{}, ctx model.Context) {
	box := ctx.TemplateBox
	t := template.New(tpl)

	var templateString string
	var err error

	templateString, err = box.String("viewtag.html")
	if err != nil {
		glog.Fatal(err)
	}

	t, err = t.Parse(templateString)
	if err != nil {
		glog.Error(err)
	}

	templateString, err = box.String("viewNewTag.html")
	if err != nil {
		glog.Fatal(err)
	}
	t.Parse(templateString)

	templateString, err = box.String("viewExistingTag.html")
	if err != nil {
		glog.Fatal(err)
	}
	t.Parse(templateString)

        for header, value := range h {
                w.Header().Set(header, value)
        }

        w.WriteHeader(status)

	err = t.Execute(w, d)
	if err != nil {
		glog.Error(err)
	}
}
