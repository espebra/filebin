package output

import (
	"io"
	"net/http"
	"encoding/json"
	"strconv"
	"html/template"

	"github.com/espebra/filebin/app/model"
)

func JSONresponse(w http.ResponseWriter, status int, h map[string]string, d interface{}, ctx model.Context) {
	dj, err := json.MarshalIndent(d, "", "    ")
	if err != nil {
		ctx.Log.Println("Unable to convert response to json: ", err)
		http.Error(w, "Failed while generating a response", http.StatusInternalServerError)
		return
	}

	for header, value := range h {
		w.Header().Set(header, value)
	}

	w.WriteHeader(status)
	ctx.Log.Println("Response status: " + strconv.Itoa(status))
	io.WriteString(w, string(dj))
}

// This function is a hack. Need to figure out a better way to do this.
func HTMLresponse(w http.ResponseWriter, tpl string, status int, h map[string]string, d interface{}, ctx model.Context) {
	box := ctx.TemplateBox
	t := template.New(tpl)

	var templateString string
	var err error

	templateString, err = box.String("viewtag.html")
	if err != nil {
		ctx.Log.Fatalln(err)
	}

	t, err = t.Parse(templateString)
	if err != nil {
		ctx.Log.Fatalln(err)
	}

	templateString, err = box.String("viewNewTag.html")
	if err != nil {
		ctx.Log.Fatalln(err)
	}
	t.Parse(templateString)

	templateString, err = box.String("viewExistingTag.html")
	if err != nil {
		ctx.Log.Fatalln(err)
	}
	t.Parse(templateString)

        for header, value := range h {
                w.Header().Set(header, value)
        }

        w.WriteHeader(status)
	ctx.Log.Println("Response status: " + strconv.Itoa(status))

	// To send multiple structs to the template
	err = t.Execute(w, map[string]interface{}{
		"Data": d,
		"Ctx": ctx,
	})
	if err != nil {
		ctx.Log.Fatalln(err)
	}
}
