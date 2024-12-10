package new

import (
	_ "embed"
	"fmt"
	"github.com/gosimple/slug"
	"github.com/manifoldco/promptui"
	"github.com/quix-labs/multipress/config"
	"github.com/quix-labs/multipress/utils"
	"github.com/urfave/cli/v2"
	"path/filepath"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:  "new",
		Usage: "Create new project",
		Args:  false,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "Force recreating file if already exists",
			},
		},
		Action: action,
	}
}

func action(c *cli.Context) error {
	// Project directory
	prompt := promptui.Prompt{
		Label:   "Where would you like to set the project location?",
		Default: "./multipress",
		Validate: func(s string) error {
			if s == "" {
				return fmt.Errorf("project location is required")
			}
			return nil
		},
	}
	projectPath, err := prompt.Run()
	if err != nil {
		fmt.Println(err)
		return err
	}

	// Check existing
	if !c.Bool("force") {
		exists, err := utils.DirectoryExists(projectPath)
		if err != nil {
			fmt.Println(err)
			return err
		}
		if exists {
			fmt.Printf("directory already exists: %s\n", projectPath)
			return fmt.Errorf("directory already exists: %s", projectPath)
		}
	}

	cfg, err := askConfig(c)
	if err != nil {
		fmt.Println(err)
		return err
	}

	if err := utils.CreateDirectoryIfNotExists(projectPath); err != nil {
		fmt.Println(err)
		return err
	}

	cfgPath := filepath.Join(projectPath, "multipress.yaml")
	if err := cfg.SaveAs(cfgPath); err != nil {
		fmt.Println("Error saving configuration:", err)
		return err
	}

	if err := cloneWpDockerfile(cfg, projectPath); err != nil {
		fmt.Println(err)
		return err
	}

	fmt.Printf("Successfully created configuration! Navigate to %s and run 'multipress deploy'.\n", projectPath)

	return nil
}

func askConfig(c *cli.Context) (*config.Config, error) {
	// Generate default configuration
	cfg := config.NewDefaultConfig()
	var err error

	// Project name
	prompt := promptui.Prompt{
		Label:   "Project Name",
		Default: cfg.Project,
		Validate: func(s string) error {
			if s == "" {
				return fmt.Errorf("project name is required")
			}
			if !slug.IsSlug(s) {
				return fmt.Errorf("project name only support [a-z], [0-9], '-' or '_'")
			}
			return nil
		},
	}
	if cfg.Project, err = prompt.Run(); err != nil {
		fmt.Println(err)
		return nil, err
	}

	// Base Domain
	prompt = promptui.Prompt{
		Label:   "Base Domain",
		Default: cfg.BaseDomain,
		Validate: func(s string) error {
			if s == "" {
				return fmt.Errorf("base domain is required")
			}
			return nil
		},
	}
	if cfg.BaseDomain, err = prompt.Run(); err != nil {
		fmt.Println(err)
		return nil, err
	}

	return cfg, nil
}

//go:embed tmpl/wordpress.Dockerfile
var wordpressTmpl string

func cloneWpDockerfile(cfg *config.Config, projectPath string) error {
	if err := utils.ParseTemplateToFile(
		wordpressTmpl,
		cfg,
		filepath.Join(projectPath, "wordpress.Dockerfile"),
	); err != nil {
		return err
	}
	return nil
}
