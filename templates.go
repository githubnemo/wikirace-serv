package main

import (
	"html/template"
	"io"
	"strings"
)

type MustTemplates struct {
	*template.Template
}

func (t *MustTemplates) MustExecuteTemplate(wr io.Writer, name string, data interface{}) {
	err := t.Template.ExecuteTemplate(wr, name, data)

	if err != nil {
		panic(err)
	}
}

func parseTemplates() (*MustTemplates, error) {
	tmp := template.New("").Funcs(map[string]interface{}{
		"format_wikiurl": func(in string) string {
			return strings.Replace(in, "_", " ", -1)
		},
	})

	tmp, err := tmp.ParseGlob("templates/*.html")

	if err != nil {
		return nil, err
	}

	return &MustTemplates{tmp}, nil
}
