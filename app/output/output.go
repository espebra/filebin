package output

import (
	"io"
	//"io/ioutil"
	"net/http"
	"encoding/json"
	"strconv"
	//"html/template"
	//"path/filepath"

        "github.com/golang/glog"
	"github.com/arschles/go-bindata-html-template"
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

func HTMLresponse(w http.ResponseWriter, tpl string, status int, h map[string]string, d interface{}) {
	//t, err := template.ParseFiles(path)
	t, err := template.New(tpl, Asset).Parse("templates/" + tpl + ".html")
	if err != nil {
		glog.Error(err)
	}

	err = t.Execute(w, d)
	if err != nil {
		glog.Error(err)
	}
}
