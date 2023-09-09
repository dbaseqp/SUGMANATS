package models

import (
	"html/template"

	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday/v2"
)

func (t UserData) IsValid() bool {
	return t.Name != ""
}

func RenderMarkdown(text string) template.HTML {
	unsafe := blackfriday.Run([]byte(text), blackfriday.WithExtensions(blackfriday.HardLineBreak))
	html := bluemonday.UGCPolicy().SanitizeBytes(unsafe)
	return template.HTML(string(html))
}