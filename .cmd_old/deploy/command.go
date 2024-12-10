package deploy

import (
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/go-sql-driver/mysql"
	"github.com/manifoldco/promptui"
	"github.com/quix-labs/multipress/config"
	"github.com/quix-labs/multipress/utils"
	"github.com/urfave/cli/v2"
	"os"
	"os/exec"
	"strings"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:   "deploy",
		Usage:  "Deploy Network + Mysql + Model",
		Args:   false,
		Action: action,
	}
}

const configPath = "multipress.yaml"

type Step struct {
	Label string
	Run   func(c *cli.Context, cfg *config.Config) error
}

var steps = []Step{
	{"Creating Docker network", createDockerNetworkIfNotExists},
	{"Configuring Caddy", configureCaddy},
	{"Deploying Caddy", deployCaddy},
	{"Configuring MySql", configureMySql},
	{"Creating Volume Directory", createVolumesDirectory},
	{"Create MySql Volume", createMysqlVolume},
	{"Deploying MySql", deployMysql},
	{"Configuring Model", configureModel},
	{"Create Model Volume", createModelVolume},
	{"Deploying Model", deployModel},
}

func action(c *cli.Context) error {
	configPath := "multipress.yaml"
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Println(err)
		return err
	}

	utils.PrintSeparator("Deployment", '═')
	for _, step := range steps {
		if err := utils.Spin(utils.SpinOptions{Label: step.Label}, func() error {
			return step.Run(c, cfg)
		}); err != nil {
			return err
		}
	}

	utils.PrintSeparator("Deployment finished", '═')
	fmt.Printf("URL: %s/wp-admin/\n", cfg.ModelUrl())
	fmt.Printf("User: %s\n", cfg.Model.Credentials.Username)
	fmt.Printf("Password: %s\n", cfg.Model.Credentials.Password)
	utils.PrintSeparator("", '═')

	return nil
}

func configureCaddy(c *cli.Context, cfg *config.Config) error {
	if cfg.Caddy != nil {
		return utils.SkippedError{Msg: "Configuration already defined"}
	}

	cfg.Caddy = config.NewDefaultCaddyConfig()
	prompt := promptui.Select{
		Label: "Select TLS Provider",
		Items: []string{"internal", "acme"},
	}

	var err error
	_, cfg.Caddy.TLSIssuer, err = prompt.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return err
	}

	return cfg.SaveAs(configPath)
}

func configureMySql(c *cli.Context, cfg *config.Config) error {
	if cfg.MySql != nil {
		return utils.SkippedError{Msg: "Configuration already defined"}
	}

	cfg.MySql = config.NewDefaultMysqlConfig()
	// You can ask here, but for simplicity, keep default auto-generated
	return cfg.SaveAs(configPath)

}

func configureModel(c *cli.Context, cfg *config.Config) error {
	if cfg.Model != nil {
		return utils.SkippedError{Msg: "Configuration already defined"}
	}

	cfg.Model = config.NewDefaultModelConfig(cfg)
	// You can ask here, but for simplicity, keep default auto-generated
	return cfg.SaveAs(configPath)
}

func createVolumesDirectory(c *cli.Context, cfg *config.Config) error {
	volumePath := cfg.VolumePath()
	if exists, err := utils.DirectoryExists(volumePath); err != nil || exists {
		if err != nil {
			return err
		}
		return utils.SkippedError{Msg: "volumes directory already exists"}
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

func createDockerNetworkIfNotExists(c *cli.Context, cfg *config.Config) error {
	checkCmd := exec.Command("docker", "network", "ls", "--filter", fmt.Sprintf("name=%s", cfg.NetworkName()), "--format", "{{.Name}}")
	existingNetworks, err := checkCmd.Output()
	if err != nil {
		return errors.New("failed to check existing networks: " + err.Error())
	}
	if strings.TrimSpace(string(existingNetworks)) == cfg.NetworkName() {
		return utils.SkippedError{Msg: "Network already exists"}
	}

	output, err := exec.Command("docker", "network", "create", cfg.NetworkName()).CombinedOutput()
	if err != nil {
		return errors.Join(err, errors.New(string(output)))
	}
	return nil
}

//go:embed tmpl/caddy.yaml.tmpl
var caddyTmpl string

func deployCaddy(c *cli.Context, cfg *config.Config) error {
	if err := utils.ParseTemplateToFile(caddyTmpl, cfg, "compose.caddy.yaml"); err != nil {
		return err
	}

	if _, err := utils.UpComposeFile("compose.caddy.yaml"); err != nil {
		return err
	}

	return nil
}

func createMysqlVolume(c *cli.Context, cfg *config.Config) error {
	volumePath := cfg.MysqlVolumePath()
	return createVolumeDirectory(cfg, volumePath)
}

//go:embed tmpl/mysql.yaml.tmpl
var mysqlTmpl string

func deployMysql(c *cli.Context, cfg *config.Config) error {
	if err := utils.ParseTemplateToFile(mysqlTmpl, cfg, "compose.mysql.yaml"); err != nil {
		return err
	}

	if _, err := utils.UpComposeFile("compose.mysql.yaml"); err != nil {
		return err
	}

	return nil
}

func createModelVolume(c *cli.Context, cfg *config.Config) error {
	volumePath := cfg.ModelVolumePath()
	return createVolumeDirectory(cfg, volumePath)
}

//go:embed tmpl/model.yaml.tmpl
var modelTmpl string

func deployModel(c *cli.Context, cfg *config.Config) error {
	if err := utils.ParseTemplateToFile(modelTmpl, cfg, "compose.model.yaml"); err != nil {
		return err
	}

	if err := bootstrapDatabaseCredentials(cfg); err != nil {
		return err
	}

	if _, err := utils.UpComposeFile("compose.model.yaml"); err != nil {
		return err
	}

	if err := installWordpress(cfg); err != nil {
		return err
	}

	return nil
}

func bootstrapDatabaseCredentials(cfg *config.Config) error {
	credentials := &cfg.Model.Credentials

	mysqlIP, err := utils.GetDockerContainerIP(cfg.MysqlContainerName())
	if err != nil {
		return err
	}

	mysqlConnector, err := mysql.NewConnector(&mysql.Config{User: "root", Passwd: cfg.MySql.RootPassword, Addr: mysqlIP})
	if err != nil {
		return fmt.Errorf("failed to connect to MySQL: %w", err)
	}
	db := sql.OpenDB(mysqlConnector)
	defer db.Close()

	statements := []string{
		fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", credentials.DBName),
		fmt.Sprintf("DROP USER IF EXISTS '%s'@'%%'", credentials.DBUser),
		fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", credentials.DBName),
		fmt.Sprintf("CREATE USER IF NOT EXISTS '%s'@'%%' IDENTIFIED BY '%s'", credentials.DBUser, credentials.DBPassword),
		fmt.Sprintf("GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'%%'", credentials.DBName, credentials.DBUser),
		"FLUSH PRIVILEGES",
	}

	for _, statement := range statements {
		//fmt.Printf("Executing SQL: %s\n", statement)
		if _, err := db.Exec(statement); err != nil {
			return fmt.Errorf("failed to execute statement %q: %w", statement, err)
		}
	}

	return nil
}

func installWordpress(cfg *config.Config) error {
	installCommands := []string{
		fmt.Sprintf(
			"wp core install --url='%s' --title='Mon site Multipress' --admin_user='%s' --admin_email='%s' --admin_password='%s' --skip-email",
			cfg.ModelUrl(), cfg.Model.Credentials.Username, cfg.Model.Credentials.Email, cfg.Model.Credentials.Password,
		),
		`echo 'php_value upload_max_filesize 2048M' >> .htaccess`,
		`echo 'php_value post_max_size 2048M' >> .htaccess`,
	}

	for _, command := range installCommands {
		if res, err := utils.ExecDockerCmd(cfg.ModelContainerName(), container.ExecOptions{User: fmt.Sprintf("%d:%d", cfg.Uid, cfg.Gid), Cmd: []string{"bash", "-c", command}}, nil); err != nil {
			return fmt.Errorf("error installing Wordpress: %v - Details: %s", err, res)
		}
	}
	return nil
}

func createVolumeDirectory(cfg *config.Config, volumePath string) error {
	if exists, err := utils.DirectoryExists(volumePath); err != nil || exists {
		if err != nil {
			return err
		}
		return utils.SkippedError{Msg: "volumes directory already exists"}
	}

	if err := utils.CreateDirectoryIfNotExists(volumePath); err != nil {
		return err
	}

	if err := os.Chown(volumePath, cfg.Uid, cfg.Gid); err != nil {
		return fmt.Errorf("failed to change ownership of the volume: %v", err)
	}
	return nil
}
