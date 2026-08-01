package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type Provider struct {
	ID            int64  `json:"id"`
	ClashName     string `json:"clash_name"`
	CollectorName string `json:"collector_name"`
	Interval      int    `json:"interval"`
	ScheduleMode  string `json:"schedule_mode"`
	PanelURL      string `json:"panel_url"`
	LandingPage   string `json:"landing_page"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	ConfigPath    string `json:"config_path,omitempty"`
}

type User struct {
	ID           int64  `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"`
}

type DB struct {
	db *sql.DB
}

func Open(path string) (*DB, error) {
	d, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	db := &DB{db: d}
	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return db, nil
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS providers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		clash_name TEXT NOT NULL UNIQUE,
		panel_url TEXT NOT NULL DEFAULT '',
		landing_page TEXT NOT NULL DEFAULT '',
		username TEXT NOT NULL DEFAULT '',
		password TEXT NOT NULL DEFAULT '',
		config_path TEXT NOT NULL DEFAULT '',
		interval INTEGER NOT NULL DEFAULT 3600,
		schedule_mode TEXT NOT NULL DEFAULT 'follow_global',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);`
	_, err := d.db.Exec(schema)
	if err != nil {
		return err
	}
	// Add collector_name column if upgrading from older schema
	d.db.Exec(`ALTER TABLE providers ADD COLUMN collector_name TEXT NOT NULL DEFAULT ''`)
	// Add schedule_mode column if upgrading from older schema
	d.db.Exec(`ALTER TABLE providers ADD COLUMN schedule_mode TEXT NOT NULL DEFAULT 'follow_global'`)
	return nil
}

func (d *DB) ListProviders() ([]Provider, error) {
	rows, err := d.db.Query(`SELECT id, clash_name, collector_name, interval, schedule_mode, panel_url, landing_page, username, config_path FROM providers ORDER BY clash_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var providers []Provider
	for rows.Next() {
		var p Provider
		if err := rows.Scan(&p.ID, &p.ClashName, &p.CollectorName, &p.Interval, &p.ScheduleMode, &p.PanelURL, &p.LandingPage, &p.Username, &p.ConfigPath); err != nil {
			return nil, err
		}
		if p.ScheduleMode == "" {
			p.ScheduleMode = "follow_global"
		}
		providers = append(providers, p)
	}
	return providers, rows.Err()
}

func (d *DB) GetProvider(id int64) (*Provider, error) {
	p := &Provider{}
	err := d.db.QueryRow(`SELECT id, clash_name, collector_name, interval, schedule_mode, panel_url, landing_page, username, password, config_path FROM providers WHERE id = ?`, id).
		Scan(&p.ID, &p.ClashName, &p.CollectorName, &p.Interval, &p.ScheduleMode, &p.PanelURL, &p.LandingPage, &p.Username, &p.Password, &p.ConfigPath)
	if err != nil {
		return nil, err
	}
	if p.ScheduleMode == "" {
		p.ScheduleMode = "follow_global"
	}
	return p, nil
}

func (d *DB) GetProviderByClashName(name string) (*Provider, error) {
	p := &Provider{}
	err := d.db.QueryRow(`SELECT id, clash_name, collector_name, interval, schedule_mode, panel_url, landing_page, username, password, config_path FROM providers WHERE clash_name = ?`, name).
		Scan(&p.ID, &p.ClashName, &p.CollectorName, &p.Interval, &p.ScheduleMode, &p.PanelURL, &p.LandingPage, &p.Username, &p.Password, &p.ConfigPath)
	if err != nil {
		return nil, err
	}
	if p.ScheduleMode == "" {
		p.ScheduleMode = "follow_global"
	}
	return p, nil
}

func (d *DB) CreateProvider(p *Provider) (int64, error) {
	res, err := d.db.Exec(`INSERT INTO providers (clash_name, collector_name, interval, schedule_mode, panel_url, landing_page, username, password, config_path) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ClashName, p.CollectorName, p.Interval, p.ScheduleMode, p.PanelURL, p.LandingPage, p.Username, p.Password, p.ConfigPath)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DB) UpdateProvider(p *Provider) error {
	_, err := d.db.Exec(`UPDATE providers SET clash_name=?, collector_name=?, interval=?, schedule_mode=?, panel_url=?, landing_page=?, username=?, password=? WHERE id=?`,
		p.ClashName, p.CollectorName, p.Interval, p.ScheduleMode, p.PanelURL, p.LandingPage, p.Username, p.Password, p.ID)
	return err
}

func (d *DB) DeleteProvider(id int64) error {
	_, err := d.db.Exec(`DELETE FROM providers WHERE id = ?`, id)
	return err
}

func (d *DB) RegisterUser(username, passwordHash string) error {
	_, err := d.db.Exec(`INSERT INTO users (username, password_hash) VALUES (?, ?)`, username, passwordHash)
	return err
}

func (d *DB) GetUser(username string) (*User, error) {
	u := &User{}
	err := d.db.QueryRow(`SELECT id, username, password_hash FROM users WHERE username = ?`, username).
		Scan(&u.ID, &u.Username, &u.PasswordHash)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (d *DB) HasUsers() (bool, error) {
	var count int
	err := d.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count)
	return count > 0, err
}

func (d *DB) GetSetting(key string) (string, error) {
	var value string
	err := d.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

func (d *DB) SetSetting(key, value string) error {
	_, err := d.db.Exec(`INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}
