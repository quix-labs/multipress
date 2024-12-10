package app

import (
	"embed"
	_ "embed"
	"fmt"
	"github.com/quix-labs/multipress/pkg/utils"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed stubs/app/*
var stubsFS embed.FS

type Application struct {
	_configDirectory string // Directory where the Application data is stored
}

func (app *Application) init() error {
	if app._configDirectory == "" {
		defaultDir, err := os.UserConfigDir()
		if err != nil {
			return err
		}
		app._configDirectory = filepath.Join(defaultDir, "multipress")
	}
	return nil
}

func (app *Application) Start() error {
	stubContent, err := app.getStub("caddy.yaml.tmpl")
	if err != nil {
		return err
	}

	fmt.Println(string(stubContent))
	return nil
}

func (app *Application) StubDirectory() string {
	return filepath.Join(app._configDirectory, "stubs")
}

func (app *Application) PublishStubs() error {
	return utils.CopyFSDir(stubsFS, "stubs/app", app.StubDirectory())
}

func (app *Application) getStub(fileName string) ([]byte, error) {
	stubsFS, err := fs.Sub(stubsFS, "stubs/app")
	if err != nil {
		return nil, err
	}

	return utils.OpenFileAcrossFilesystem(
		fileName,
		os.DirFS(app.StubDirectory()),
		stubsFS,
	)
}

var app *Application

func LoadApp(directory string) error {
	app = &Application{
		_configDirectory: directory,
	}
	return app.init()
}
func GetApplication() *Application {
	return app
}
