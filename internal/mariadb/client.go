package mariadb

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

type Client struct {
	db *sql.DB
}

func SanitizeIdentifier(name string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	cleaned := re.ReplaceAllString(name, "_")
	return strings.Trim(cleaned, "_")
}

func NewClient(host string, port int, user, password string) (*Client, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?timeout=5s", user, password, host, port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open mysql connection: %w", err)
	}

	return &Client{db: db}, nil
}

func (c *Client) Ping() error {
	return c.db.Ping()
}

func (c *Client) Close() error {
	return c.db.Close()
}

func (c *Client) CreateDatabaseAndUser(dbName, username, password string) error {
	cleanDB := SanitizeIdentifier(dbName)
	cleanUser := SanitizeIdentifier(username)

	tx, err := c.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;", cleanDB)); err != nil {
		return fmt.Errorf("create database: %w", err)
	}

	if _, err := tx.Exec(fmt.Sprintf("CREATE USER IF NOT EXISTS '%s'@'%%' IDENTIFIED BY '%s';", cleanUser, password)); err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	if _, err := tx.Exec(fmt.Sprintf("GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'%%';", cleanDB, cleanUser)); err != nil {
		return fmt.Errorf("grant privileges: %w", err)
	}

	if _, err := tx.Exec("FLUSH PRIVILEGES;"); err != nil {
		return fmt.Errorf("flush privileges: %w", err)
	}

	return tx.Commit()
}

func (c *Client) DropDatabaseAndUser(dbName, username string) error {
	cleanDB := SanitizeIdentifier(dbName)
	cleanUser := SanitizeIdentifier(username)

	_, _ = c.db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`;", cleanDB))
	_, _ = c.db.Exec(fmt.Sprintf("DROP USER IF EXISTS '%s'@'%%';", cleanUser))
	_, _ = c.db.Exec("FLUSH PRIVILEGES;")
	return nil
}
