package utils

import (
	"os"
	"text/template"
)

func ParseTemplateToFile(tpl string, data interface{}, filepath string) error {
	tmpl, err := template.New("").Parse(tpl)
	if err != nil {
		return err
	}

	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return err
	}
	return nil
}
