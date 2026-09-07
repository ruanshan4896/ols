package mariadb

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

type ContainerClient struct {
	container string
	rootPass  string
}

func NewContainerClient(container, rootPass string) *ContainerClient {
	return &ContainerClient{
		container: container,
		rootPass:  rootPass,
	}
}

func SanitizeIdentifier(name string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	cleaned := re.ReplaceAllString(name, "_")
	return strings.Trim(cleaned, "_")
}

func (c *ContainerClient) ExecSQL(query string) error {
	cmd := exec.Command("docker", "exec", "-i", c.container, "mariadb", "-uroot", "-p"+c.rootPass, "-e", query)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mariadb exec: %s (%w)", string(out), err)
	}
	return nil
}

func (c *ContainerClient) CreateDatabaseAndUser(dbName, username, password string) error {
	cleanDB := SanitizeIdentifier(dbName)
	cleanUser := SanitizeIdentifier(username)

	sql := fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci; "+
			"CREATE USER IF NOT EXISTS '%s'@'%%' IDENTIFIED BY '%s'; "+
			"GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'%%'; "+
			"FLUSH PRIVILEGES;",
		cleanDB, cleanUser, password, cleanDB, cleanUser,
	)

	return c.ExecSQL(sql)
}

func (c *ContainerClient) DropDatabaseAndUser(dbName, username string) error {
	cleanDB := SanitizeIdentifier(dbName)
	cleanUser := SanitizeIdentifier(username)

	sql := fmt.Sprintf(
		"DROP DATABASE IF EXISTS `%s`; "+
			"DROP USER IF EXISTS '%s'@'%%'; "+
			"FLUSH PRIVILEGES;",
		cleanDB, cleanUser,
	)

	return c.ExecSQL(sql)
}
