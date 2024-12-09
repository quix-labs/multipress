package doctor

import (
	"errors"
	"fmt"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/urfave/cli/v2"
	"os"
	"os/exec"
	"strings"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:  "doctor",
		Usage: "Verify system has all requirements",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Only check dependencies, ignore installing",
			},
		},
		Action: action,
	}
}

type check struct {
	name        string
	description string
	check       func(*cli.Context) (bool, error)
	resolve     func(*cli.Context) error
	required    bool
}

var checks = []*check{
	{
		name:        "OS",
		description: "Debian, Ubuntu",
		check: func(context *cli.Context) (bool, error) {
			return isSupportedOS(), nil
		},
		required: true,
	},
	{
		name:        "Docker",
		description: "Check docker installed",
		check: func(context *cli.Context) (bool, error) {
			return isCommandAvailable("docker"), nil
		},
		resolve: func(context *cli.Context) error {
			return installAptDependencies("docker-ce", "docker-ce-cli")
		},
	},
	{
		name:        "Docker Compose",
		description: "Check docker compose installed",
		check: func(context *cli.Context) (bool, error) {
			return isDockerComposeAvailable(), nil
		},
		resolve: func(context *cli.Context) error {
			return installAptDependencies("docker-compose-plugin")
		},
	},
	{
		name:        "Zip",
		description: "Check zip installed",
		check: func(context *cli.Context) (bool, error) {
			return isCommandAvailable("zip"), nil
		},
		resolve: func(context *cli.Context) error {
			return installAptDependencies("zip", "unzip")
		},
	},
}

func isSupportedOS() bool {
	cmd := exec.Command("sh", "-c", "cat /etc/os-release | grep -E '^(ID=|ID_LIKE=)'")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), "debian") || strings.Contains(string(output), "ubuntu")
}

func isCommandAvailable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

func isDockerComposeAvailable() bool {
	cmd := exec.Command("docker", "compose", "version")
	output, err := cmd.Output()
	return err == nil && strings.Contains(string(output), "version")
}

func installAptDependencies(deps ...string) error {
	cmd := exec.Command("apt-get", append([]string{"install", "-y"}, deps...)...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return errors.Join(err, errors.New(string(output)))
	}
	return nil
}
func action(c *cli.Context) error {
	t := table.NewWriter()
	t.SetStyle(table.StyleLight)
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Name", "Description", "Check", "Resolvable"})

	for _, check := range checks {
		valid, err := check.check(c)
		if err != nil {
			fmt.Println(err)
			return err
		}

		var validString = text.FgRed.Sprintf("NO")
		if valid {
			validString = text.FgGreen.Sprintf("YES")
		}

		var resolvableString = "---"

		if !valid && check.resolve != nil {
			resolvableString = text.FgGreen.Sprintf("YES")
		} else if !valid && check.resolve == nil {
			resolvableString = text.FgRed.Sprintf("NO")
		}

		t.AppendRow([]interface{}{check.name, check.description, validString, resolvableString})
		t.AppendSeparator()

	}
	t.Render()

	if c.Bool("dry-run") {
		return nil
	}

	// Resolve missing dependencies
	// TODO OPTIMISE
	for _, check := range checks {
		valid, err := check.check(c)
		if err != nil || valid == true || check.resolve == nil {
			continue
		}

		fmt.Printf("Resolving: %s...\n", check.name)
		err = check.resolve(c)
		if err == nil {
			fmt.Printf("Resolved: %s\n", check.name)
		} else {
			fmt.Printf("Failed to resolve: %s\n", check.name)
			fmt.Println(err)
		}
	}
	return nil
}
