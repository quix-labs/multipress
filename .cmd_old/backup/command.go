package backup

import (
	_ "embed"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/quix-labs/multipress/config"
	"github.com/quix-labs/multipress/utils"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"
	"io"
	"os"
	"path/filepath"
	"time"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:   "backup",
		Usage:  "Generate backups",
		Action: action,
	}
}

const folderDataFormat = "20060102_150405"

type InstanceStep struct {
	Label string
	Run   func(c *cli.Context, cfg *config.Config, identifier string, start time.Time) error
}

type Step struct {
	Label string
	Run   func(c *cli.Context, cfg *config.Config, start time.Time) error
}

var preSteps = []Step{
	{"Create backups directory", createBackupsDirectory},
	{"Create backups/date directory", createBackupsDateDirectory},
	{"Deploy backup server", deployBackupServer},
}

var steps = []InstanceStep{
	{"Create backup/date/instance directories", createInstanceBackupDir},
	{"Generate SQL Dumps", dumpSqlInstance},
	{"Copy Sources", copyInstanceSource},
	{"Copy compose.yaml files", copyInstanceCompose},
	{"Compress Backups", createArchiveInstance},
	{"Delete backup/date/instance directories", deleteInstanceBackupDir},
}

func action(c *cli.Context) error {
	configPath := "multipress.yaml"
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Println(err)
		return err
	}

	startDate := time.Now()

	utils.PrintSeparator("Pre-Steps", '═')
	for _, step := range preSteps {
		if err := utils.Spin(utils.SpinOptions{Label: step.Label}, func() error {
			return step.Run(c, cfg, startDate)
		}); err != nil {
			return err
		}
	}

	// Replicate instances, // Run all steps in parallel
	utils.PrintSeparator("Backup instances", '═')
	for _, step := range steps {
		if err := utils.Spin(utils.SpinOptions{Label: step.Label}, func() error {
			var g errgroup.Group

			for identifier, _ := range cfg.Instances.Credentials {
				identifier := identifier // Important local copy
				g.Go(func() error {
					if err := step.Run(c, cfg, identifier, startDate); err != nil {
						fmt.Println(err) // TODO HANDLE MULTIPLE PROGRESS PARALLEL FAILABLE
					}
					return nil
				})
			}

			return g.Wait()
		}); err != nil {
			return err
		}
	}

	utils.PrintSeparator("Backup finished", '═')
	fmt.Printf("URL: %s/%s\n", cfg.BackupsUrl(), startDate.Format(folderDataFormat))
	utils.PrintSeparator("", '═')

	return nil
}

func createBackupsDirectory(c *cli.Context, cfg *config.Config, start time.Time) error {
	volumePath := cfg.BackupsPath()
	if exists, err := utils.DirectoryExists(volumePath); err != nil || exists {
		if err != nil {
			return err
		}
		return utils.SkippedError{Msg: "backups directory already exists"}
	}

	if err := utils.CreateDirectoryIfNotExists(volumePath); err != nil {
		return err
	}

	err := os.Chown(volumePath, cfg.Uid, cfg.Gid)
	if err != nil {
		return fmt.Errorf("failed to change ownership of the volume: %v", err)
	}
	return nil
}
func createBackupsDateDirectory(c *cli.Context, cfg *config.Config, start time.Time) error {
	volumePath := filepath.Join(cfg.BackupsPath(), start.Format(folderDataFormat))
	if exists, err := utils.DirectoryExists(volumePath); err != nil || exists {
		if err != nil {
			return err
		}
		return utils.SkippedError{Msg: "backups directory already exists"}
	}

	if err := utils.CreateDirectoryIfNotExists(volumePath); err != nil {
		return err
	}

	err := os.Chown(volumePath, cfg.Uid, cfg.Gid)
	if err != nil {
		return fmt.Errorf("failed to change ownership of the volume: %v", err)
	}
	return nil
}

func createInstanceBackupDir(c *cli.Context, cfg *config.Config, identifier string, start time.Time) error {
	backupDir := filepath.Join(cfg.BackupsPath(), start.Format(folderDataFormat), identifier)
	if exists, err := utils.DirectoryExists(backupDir); err != nil || exists {
		if err != nil {
			return err
		}
		return utils.SkippedError{Msg: "backups directory already exists"}
	}

	if err := utils.CreateDirectoryIfNotExists(backupDir); err != nil {
		return err
	}

	err := os.Chown(backupDir, cfg.Uid, cfg.Gid)
	if err != nil {
		return fmt.Errorf("failed to change ownership of the volume: %v", err)
	}
	return nil
}

func dumpSqlInstance(c *cli.Context, cfg *config.Config, identifier string, start time.Time) error {
	credentials, exists := cfg.Instances.Credentials[identifier]
	if !exists {
		return errors.New("instance credentials does not exist")
	}

	dumpPath := filepath.Join(cfg.BackupsPath(), start.Format(folderDataFormat), identifier, "dump.sql")
	output, err := utils.ExecDockerCmd(cfg.MysqlContainerName(), container.ExecOptions{
		Env: []string{fmt.Sprintf(`MYSQL_PWD=%s`, cfg.MySql.RootPassword)},
		Cmd: []string{"mysqldump", "-u", "root", credentials.DBName},
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to execute mysqldump: %w", err)
	}

	if err := os.WriteFile(dumpPath, []byte(output), 0644); err != nil {
		return fmt.Errorf("failed to write dump to file: %w", err)
	}
	return nil
}

func copyInstanceSource(c *cli.Context, cfg *config.Config, identifier string, start time.Time) error {
	sourcesDumpPath := filepath.Join(cfg.BackupsPath(), start.Format(folderDataFormat), identifier, "sources")
	return utils.CopyDirectory(cfg.InstanceVolumePath(identifier), sourcesDumpPath)
}

func copyInstanceCompose(c *cli.Context, cfg *config.Config, identifier string, start time.Time) error {
	srcComposePath := fmt.Sprintf("compose.%s.yaml", identifier)
	dstComposePath := filepath.Join(cfg.BackupsPath(), start.Format(folderDataFormat), identifier, "compose.yaml")
	return utils.CopyFile(srcComposePath, dstComposePath)
}

func createArchiveInstance(c *cli.Context, cfg *config.Config, identifier string, start time.Time) error {
	instancePath := filepath.Join(cfg.BackupsPath(), start.Format(folderDataFormat), identifier)
	zipPath := filepath.Join(cfg.BackupsPath(), start.Format(folderDataFormat), identifier+".tar.gz")

	reader, err := utils.TarGzDirectory(instancePath)
	if err != nil {
		return err
	}

	archiveFile, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("failed to create archive file %s: %w", zipPath, err)
	}
	defer archiveFile.Close()

	if _, err := io.Copy(archiveFile, reader); err != nil {
		return fmt.Errorf("failed to compress: %w", err)
	}

	return nil
}

func deleteInstanceBackupDir(c *cli.Context, cfg *config.Config, identifier string, start time.Time) error {
	backupDir := filepath.Join(cfg.BackupsPath(), start.Format(folderDataFormat), identifier)
	if exists, err := utils.DirectoryExists(backupDir); err != nil || !exists {
		if err != nil {
			return err
		}
		return utils.SkippedError{Msg: "backups directory not exists"}
	}

	return utils.RemoveDirectory(backupDir, true)
}

//go:embed tmpl/backup.yaml.tmpl
var backupTmpl string

func deployBackupServer(c *cli.Context, cfg *config.Config, start time.Time) error {
	if err := utils.ParseTemplateToFile(backupTmpl, cfg, "compose.backup.yaml"); err != nil {
		return err
	}

	if _, err := utils.UpComposeFile("compose.backup.yaml"); err != nil {
		return err
	}

	return nil
}
