package main

import (
	"html/template"
	"strings"
)

func parseTemplates() (err error) {
	templates = template.New("").Funcs(map[string]interface{}{
		"format_wikiurl": func(in string) string {
			return strings.Replace(in, "_", " ", -1)
		},
	})

	templates, err = templates.ParseGlob("templates/*.html")

	return err
}


