package main

import (
	"html/template"
	"io"
	"strings"
	"bytes"
)

type MustTemplates struct {
	*template.Template
}

// TODO: replace with template.Must
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
		"is_winner": func(g *Game, p Player) bool {
			isWinner, _ := g.EvaluateWinner(&p)
			return isWinner
		},
		"is_temporary_winner": func(g *Game, p Player) bool {
			_, isTemp := g.EvaluateWinner(&p)
			return isTemp
		},
	})

	tmp, err := tmp.ParseGlob("templates/*.html")

	if err != nil {
		return nil, err
	}

	return &MustTemplates{tmp}, nil
}

// Passed to the wikis package to render the wiki page in wiki.ServePage().
func WikiPageRenderer(header, content template.HTML) (string, error) {
	buf := bytes.NewBuffer([]byte{})

	err := templates.ExecuteTemplate(buf, "wiki.html", struct {
		Header  template.HTML
		Content template.HTML
	}{template.HTML(header), template.HTML(content)})

	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
