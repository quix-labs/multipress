package templates

import (
	"errors"
	"github.com/quix-labs/multipress/pkg/app"
)

type Template struct {
	_directory string
	Name       string
	Type       string // Git, Local, Internal
	Uri        string // Depends on type
}

// Templates structure to manage template configurations.
type Templates struct {
	_directory string
	Name       string
	Type       string // Git, Local, Internal
	Uri        string // Depends on type
}

func LoadTemplates() error {
	currentApp := app.GetApplication()
	if currentApp == nil {
		return errors.New("application not initialized")
	}
	return nil
}
