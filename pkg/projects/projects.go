package projects

import (
	"errors"
	"github.com/quix-labs/multipress/pkg/app"
	"github.com/quix-labs/multipress/pkg/templates"
)

type Project struct {
	_workingDirectory string
	Template          *templates.Template
	Name              string
	Volumes           []string
	Deployments       []string
}

// Projects structure to manage project configurations.
type Projects struct {
	_directory string
}

func LoadProjects() error {
	currentApp := app.GetApplication()
	if currentApp == nil {
		return errors.New("application not initialized")
	}
	return nil
}
