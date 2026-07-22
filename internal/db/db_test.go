package db

import (
	"path/filepath"
	"testing"
)

func openTestDB(t *testing.T) *DB {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	d, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

func TestOpen_CreatesTables(t *testing.T) {
	d := openTestDB(t)

	var tables []string
	rows, err := d.db.Query(`SELECT name FROM sqlite_master WHERE type='table' ORDER BY name`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		rows.Scan(&name)
		tables = append(tables, name)
	}

	expected := map[string]bool{"providers": true, "users": true, "settings": true}
	for _, name := range tables {
		delete(expected, name)
	}
	if len(expected) > 0 {
		t.Errorf("missing tables: %v", expected)
	}
}

func TestCreateAndListProviders(t *testing.T) {
	d := openTestDB(t)

	p := &Provider{
		ClashName:     "my-provider",
		CollectorName: "my-collector",
		Interval:      1800,
		PanelURL:      "https://panel.example.com",
		LandingPage:   "https://landing.example.com",
		Username:      "user",
		Password:      "pass",
	}

	id, err := d.CreateProvider(p)
	if err != nil {
		t.Fatal(err)
	}
	if id <= 0 {
		t.Errorf("expected positive id, got %d", id)
	}

	list, err := d.ListProviders()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(list))
	}

	got := list[0]
	if got.ClashName != "my-provider" {
		t.Errorf("ClashName: got %q", got.ClashName)
	}
	if got.CollectorName != "my-collector" {
		t.Errorf("CollectorName: got %q", got.CollectorName)
	}
	if got.Password != "" {
		t.Errorf("ListProviders should not return password, got %q", got.Password)
	}
}

func TestGetProvider(t *testing.T) {
	d := openTestDB(t)

	p := &Provider{
		ClashName:     "get-test",
		CollectorName: "coll",
		Password:      "secret123",
	}
	id, err := d.CreateProvider(p)
	if err != nil {
		t.Fatal(err)
	}

	got, err := d.GetProvider(id)
	if err != nil {
		t.Fatal(err)
	}
	if got.ClashName != "get-test" {
		t.Errorf("ClashName: got %q", got.ClashName)
	}
	if got.Password != "secret123" {
		t.Errorf("Password: got %q", got.Password)
	}
}

func TestGetProvider_NotFound(t *testing.T) {
	d := openTestDB(t)
	_, err := d.GetProvider(999)
	if err == nil {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestUpdateProvider(t *testing.T) {
	d := openTestDB(t)

	p := &Provider{ClashName: "before", CollectorName: "c1", Password: "old"}
	id, _ := d.CreateProvider(p)

	p.ID = id
	p.ClashName = "after"
	p.CollectorName = "c2"
	p.Password = "new"
	if err := d.UpdateProvider(p); err != nil {
		t.Fatal(err)
	}

	got, _ := d.GetProvider(id)
	if got.ClashName != "after" {
		t.Errorf("ClashName: got %q", got.ClashName)
	}
	if got.CollectorName != "c2" {
		t.Errorf("CollectorName: got %q", got.CollectorName)
	}
	if got.Password != "new" {
		t.Errorf("Password: got %q", got.Password)
	}
}

func TestDeleteProvider(t *testing.T) {
	d := openTestDB(t)

	p := &Provider{ClashName: "delete-me", CollectorName: "c"}
	id, _ := d.CreateProvider(p)

	if err := d.DeleteProvider(id); err != nil {
		t.Fatal(err)
	}

	list, _ := d.ListProviders()
	if len(list) != 0 {
		t.Errorf("expected 0 providers after delete, got %d", len(list))
	}
}

func TestUniqueClashName(t *testing.T) {
	d := openTestDB(t)

	d.CreateProvider(&Provider{ClashName: "dup", CollectorName: "c1"})
	_, err := d.CreateProvider(&Provider{ClashName: "dup", CollectorName: "c2"})
	if err == nil {
		t.Fatal("expected error for duplicate clash_name")
	}
}

func TestRegisterAndGetUser(t *testing.T) {
	d := openTestDB(t)

	if err := d.RegisterUser("admin", "hash-of-password"); err != nil {
		t.Fatal(err)
	}

	u, err := d.GetUser("admin")
	if err != nil {
		t.Fatal(err)
	}
	if u.Username != "admin" {
		t.Errorf("Username: got %q", u.Username)
	}
	if u.PasswordHash != "hash-of-password" {
		t.Errorf("PasswordHash: got %q", u.PasswordHash)
	}
}

func TestGetUser_NotFound(t *testing.T) {
	d := openTestDB(t)
	_, err := d.GetUser("nobody")
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
}

func TestDuplicateUser(t *testing.T) {
	d := openTestDB(t)
	d.RegisterUser("dup", "hash1")
	err := d.RegisterUser("dup", "hash2")
	if err == nil {
		t.Fatal("expected error for duplicate username")
	}
}

func TestHasUsers(t *testing.T) {
	d := openTestDB(t)

	has, err := d.HasUsers()
	if err != nil {
		t.Fatal(err)
	}
	if has {
		t.Error("expected false when no users exist")
	}

	d.RegisterUser("admin", "hash")
	has, _ = d.HasUsers()
	if !has {
		t.Error("expected true after user registration")
	}
}

func TestSettings(t *testing.T) {
	d := openTestDB(t)

	val, err := d.GetSetting("nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if val != "" {
		t.Errorf("expected empty for missing key, got %q", val)
	}

	if err := d.SetSetting("theme", "dark"); err != nil {
		t.Fatal(err)
	}
	val, _ = d.GetSetting("theme")
	if val != "dark" {
		t.Errorf("got %q, want %q", val, "dark")
	}

	if err := d.SetSetting("theme", "light"); err != nil {
		t.Fatal(err)
	}
	val, _ = d.GetSetting("theme")
	if val != "light" {
		t.Errorf("after update: got %q, want %q", val, "light")
	}
}

func TestMultipleSettings(t *testing.T) {
	d := openTestDB(t)

	d.SetSetting("key1", "val1")
	d.SetSetting("key2", "val2")

	v1, _ := d.GetSetting("key1")
	v2, _ := d.GetSetting("key2")
	if v1 != "val1" || v2 != "val2" {
		t.Errorf("got %q / %q, want val1 / val2", v1, v2)
	}
}

func TestProviderInterval(t *testing.T) {
	d := openTestDB(t)

	p := &Provider{ClashName: "interval-test", CollectorName: "c", Interval: 7200}
	id, _ := d.CreateProvider(p)

	got, _ := d.GetProvider(id)
	if got.Interval != 7200 {
		t.Errorf("interval: got %d, want 7200", got.Interval)
	}
}
