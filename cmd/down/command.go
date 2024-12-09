package down

import (
	_ "embed"
	"fmt"
	"github.com/quix-labs/multipress/config"
	"github.com/quix-labs/multipress/utils"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"
	"path/filepath"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:   "down",
		Usage:  "Down all project resources",
		Action: action,
	}
}

type Step struct {
	Label string
	Run   func(c *cli.Context, cfg *config.Config) error
}

var steps = []Step{
	{"Stop all containers", stopAllContainers},
}

func action(c *cli.Context) error {
	configPath := "multipress.yaml"
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Println(err)
		return err
	}
	for _, step := range steps {
		if err := utils.Spin(utils.SpinOptions{Label: step.Label}, func() error {
			return step.Run(c, cfg)
		}); err != nil {
			return err
		}
	}
	return nil
}

func stopAllContainers(c *cli.Context, cfg *config.Config) error {
	pattern := filepath.Join("compose.*.yaml")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to glob files: %w", err)
	}

	if len(files) == 0 {
		return utils.SkippedError{Msg: "No compose files found"}
	}

	// Process each file in parallel
	var g errgroup.Group
	for _, file := range files {
		file := file // Important keep copy
		g.Go(func() error {
			_, err := utils.DownComposeFile(file)
			return err
		})
	}
	return g.Wait()
}
