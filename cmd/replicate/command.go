package replicate

import (
	"database/sql"
	_ "embed"
	"encoding/csv"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/go-sql-driver/mysql"
	"github.com/quix-labs/multipress/config"
	"github.com/quix-labs/multipress/utils"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:      "replicate",
		Usage:     "Replicate model onto multiple instances",
		Action:    action,
		ArgsUsage: "<count>",
	}
}

const dumpPath = "model_dump.sql"
const configPath = "multipress.yaml"
const credentialCsvPath = "instance-credentials.csv"

type Step struct {
	Label string
	Run   func(c *cli.Context, cfg *config.Config) error
}

type InstanceStep struct {
	Label string
	Run   func(c *cli.Context, cfg *config.Config, identifier string) error
}

var preSteps = []Step{
	{"Initialize instances configuration", initializeInstancesConfiguration},
	{"Dump model database", dumpModelDatabase},
	{"Create credentials.csv", createCsvCredentials},
}

var steps = []InstanceStep{
	{"Configuring instance", configureInstance},
	{"Cloning Model Volume", cloneModelVolumeInstance},
	{"Bootstrapping database", bootstrapInstanceDatabase},
	{"Deploying Instance", deployInstance},
	{"Override configuration", overrideWordpress},
}

var postSteps = []Step{
	{"Delete model dump", deleteModelDump},
}

func action(c *cli.Context) error {
	configPath := "multipress.yaml"
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Println(err)
		return err
	}

	// Parse arguments
	if c.Args().Len() != 1 {
		fmt.Println("Usage: replicate <count>")
		return errors.New("invalid argument")
	}

	countArg := c.Args().First()
	if countArg == "" {
		return errors.New("count argument not defined")
	}
	count, err := strconv.Atoi(countArg)
	if err != nil {
		return err
	}

	utils.PrintSeparator("Pre-Steps", '═')
	for _, step := range preSteps {
		if err := utils.Spin(utils.SpinOptions{Label: step.Label}, func() error {
			return step.Run(c, cfg)
		}); err != nil {
			return err
		}
	}

	// Pre-generate instance identifiers
	var identifiers = make([]string, count)
	for i := 0; i < count; i++ {
		identifiers[i] = cfg.Instances.NextIdentifier()
	}

	utils.PrintSeparator("Steps", '═')
	// Replicate instances // Parallel
	for _, step := range steps {
		if err := utils.Spin(utils.SpinOptions{Label: step.Label}, func() error {
			var g errgroup.Group
			for _, identifier := range identifiers {
				identifier := identifier // Important keep copy
				g.Go(func() (err error) {
					if err := step.Run(c, cfg, identifier); err != nil && errors.As(err, &utils.SkippedError{}) {
						return err // TODO CLEAN TERMINAL MULTI SPINNER
					}
					return nil
				})
			}
			return g.Wait()
		}); err != nil {
			return err
		}
	}

	utils.PrintSeparator("Post-Steps", '═')
	for _, step := range postSteps {
		if err := utils.Spin(utils.SpinOptions{Label: step.Label}, func() error {
			return step.Run(c, cfg)
		}); err != nil {
			return err
		}
	}

	utils.PrintSeparator("Access", '═')
	for _, identifier := range identifiers {
		fmt.Printf("URL: %s - Username: %s - Password: %s\n", cfg.InstanceUrl(identifier), cfg.Instances.Credentials[identifier].Username, cfg.Instances.Credentials[identifier].Password)
	}
	return nil
}

func initializeInstancesConfiguration(c *cli.Context, cfg *config.Config) error {
	if cfg.Instances != nil {
		return utils.SkippedError{Msg: "instances already initialized"}
	}
	cfg.Instances = config.NewDefaultInstancesConfig(cfg)
	// Ask for memory if needed
	return nil
}

func dumpModelDatabase(c *cli.Context, cfg *config.Config) error {
	output, err := utils.ExecDockerCmd(cfg.MysqlContainerName(), container.ExecOptions{
		Env: []string{fmt.Sprintf(`MYSQL_PWD=%s`, cfg.MySql.RootPassword)},
		Cmd: []string{"mysqldump", "-u", "root", "model"},
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to execute mysqldump: %w", err)
	}

	if err := os.WriteFile(dumpPath, []byte(output), 0644); err != nil {
		return fmt.Errorf("failed to write dump to file: %w", err)
	}
	return nil
}

func createCsvCredentials(c *cli.Context, cfg *config.Config) error {
	if utils.FileExists(credentialCsvPath) {
		return utils.SkippedError{Msg: "csv already exists"}
	}

	file, err := os.Create(credentialCsvPath)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("failed to close CSV file: %w", err)
	}

	return appendCsvCredential([]string{"URL", "Username", "Password", "DB Username", "DB Password", "DB Database"})

}

var instanceCfgMutex = new(sync.Mutex)

func configureInstance(c *cli.Context, cfg *config.Config, identifier string) error {
	instanceCfgMutex.Lock()
	defer instanceCfgMutex.Unlock()

	credentials := config.NewDefaultInstanceCredentialConfig(cfg, identifier)
	cfg.Instances.Credentials[identifier] = *credentials

	if err := cfg.SaveAs(configPath); err != nil {
		return err
	}
	return appendCsvCredential([]string{
		cfg.InstanceUrl(identifier),
		credentials.Username,
		credentials.Password,
		credentials.DBUser,
		credentials.DBPassword,
		credentials.DBName,
	})
}

func cloneModelVolumeInstance(c *cli.Context, cfg *config.Config, identifier string) error {
	modelVolumePath := filepath.Join("./volumes", "model")
	instanceVolumePath := filepath.Join("./volumes", identifier)

	// Check target folder not exists
	if exists, err := utils.DirectoryExists(instanceVolumePath); err != nil || exists {
		if err != nil {
			return err
		}
		return utils.SkippedError{Msg: "volumes directory already exists"}
	}

	if err := utils.CopyDirectory(modelVolumePath, instanceVolumePath); err != nil {
		return err
	}

	if err := os.Chown(modelVolumePath, cfg.Uid, cfg.Gid); err != nil {
		return fmt.Errorf("failed to change ownership of the volume: %v", err)
	}
	return nil
}

func bootstrapInstanceDatabase(c *cli.Context, cfg *config.Config, identifier string) error {
	credentials, exists := cfg.Instances.Credentials[identifier]
	if !exists {
		return errors.New("instance credentials does not exist")
	}

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

	// Create user + database
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

	// Create instance connection
	mysqlInstanceConnector, err := mysql.NewConnector(&mysql.Config{User: "root", Passwd: cfg.MySql.RootPassword, Addr: mysqlIP, DBName: credentials.DBName})
	if err != nil {
		return fmt.Errorf("failed to connect to instance database: %w", err)
	}
	dbInstance := sql.OpenDB(mysqlInstanceConnector)
	defer dbInstance.Close()

	// Import dump
	dumpData, err := os.ReadFile(dumpPath)
	if err != nil {
		return fmt.Errorf("failed to read dump file: %w", err)
	}

	if _, err = utils.ExecDockerCmd(cfg.MysqlContainerName(), container.ExecOptions{
		Env: []string{fmt.Sprintf(`MYSQL_PWD=%s`, cfg.MySql.RootPassword)},
		Cmd: []string{"mysql", "-u", "root", credentials.DBName},
	}, dumpData); err != nil {
		return err
	}

	// Replace database entries
	wordpressHost := cfg.InstanceUrl(identifier)
	statements = []string{
		fmt.Sprintf("UPDATE wp_users SET user_pass = MD5('%s'), user_url='%s', user_login='%s', user_nicename='%s', display_name='%s', user_email='%s' WHERE wp_users.ID = 1;",
			credentials.Password, wordpressHost, credentials.Username, credentials.Username, credentials.Username, credentials.Email),
		fmt.Sprintf("UPDATE wp_options SET option_value = '%s' WHERE wp_options.option_name = 'siteurl' OR wp_options.option_name = 'home';", wordpressHost),
		fmt.Sprintf("UPDATE wp_options SET option_value = '%s' WHERE wp_options.option_name = 'admin_email';", credentials.Email),
	}

	for _, statement := range statements {
		//fmt.Printf("Executing SQL: %s\n", statement)
		if _, err := dbInstance.Exec(statement); err != nil {
			return fmt.Errorf("failed to execute statement %q: %w", statement, err)
		}
	}
	return nil
}

//go:embed tmpl/instance.yaml.tmpl
var instanceTmpl string

type InstanceTmplData struct {
	Identifier  string
	Config      *config.Config
	Credentials config.CredentialsConfig
}

func deployInstance(c *cli.Context, cfg *config.Config, identifier string) error {
	data := InstanceTmplData{
		Identifier:  identifier,
		Config:      cfg,
		Credentials: cfg.Instances.Credentials[identifier],
	}

	composeFilename := fmt.Sprintf("compose.%s.yaml", identifier)

	if err := utils.ParseTemplateToFile(instanceTmpl, data, composeFilename); err != nil {
		return err
	}
	if _, err := utils.UpComposeFile(composeFilename); err != nil {
		return err
	}

	return nil
}

func overrideWordpress(c *cli.Context, cfg *config.Config, identifier string) error {
	installCommands := []string{
		fmt.Sprintf("wp search-replace '%s', '%s'", cfg.ModelUrl(), cfg.InstanceUrl(identifier)),
	}

	for _, command := range installCommands {
		if res, err := utils.ExecDockerCmd(cfg.InstanceContainerName(identifier), container.ExecOptions{User: fmt.Sprintf("%d:%d", cfg.Uid, cfg.Gid), Cmd: []string{"bash", "-c", command}}, nil); err != nil {
			return fmt.Errorf("error override Wordpress: %v - Details: %s", err, res)
		}
	}

	return nil
}

func deleteModelDump(c *cli.Context, cfg *config.Config) error {
	if !utils.FileExists(dumpPath) {
		return utils.SkippedError{Msg: "dumpModelDatabase not found"}
	}
	return utils.RemoveFile(dumpPath)
}

var csvLock = new(sync.Mutex)

func appendCsvCredential(data []string) error {
	csvLock.Lock()
	defer csvLock.Unlock()

	file, err := os.OpenFile(credentialCsvPath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if err := writer.Write(data); err != nil {
		return fmt.Errorf("failed to write data to CSV file: %w", err)
	}

	return nil
}
