package output

import (
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"strconv"

	"github.com/espebra/filebin/app/model"
)

func JSONresponse(w http.ResponseWriter, status int, d interface{}, ctx model.Context) {
	dj, err := json.MarshalIndent(d, "", "    ")
	if err != nil {
		ctx.Log.Println("Unable to convert response to json: ", err)
		http.Error(w, "Failed while generating a response", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(status)
	ctx.Log.Println("Response status: " + strconv.Itoa(status))
	io.WriteString(w, string(dj))
}

// This function is a hack. Need to figure out a better way to do this.
func HTMLresponse(w http.ResponseWriter, tpl string, status int, d interface{}, ctx model.Context) {
	box := ctx.TemplateBox
	t := template.New(tpl)

	var templateString string
	var err error

	templateString, err = box.String(tpl + ".html")
	if err != nil {
		ctx.Log.Fatalln(err)
	}
	t, err = t.Parse(templateString)
	if err != nil {
		ctx.Log.Fatalln(err)
	}

	w.WriteHeader(status)
	ctx.Log.Println("Response status: " + strconv.Itoa(status))

	// To send multiple structs to the template
	err = t.Execute(w, map[string]interface{}{
		"Data": d,
		"Ctx":  ctx,
	})
	if err != nil {
		ctx.Log.Panicln(err)
	}
}
