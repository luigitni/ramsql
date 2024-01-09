package ramsql

import (
	"database/sql"
	"sync"
	"testing"

	"github.com/proullon/ramsql/engine/log"
)

func TestTransaction(t *testing.T) {

	db, err := sql.Open("ramsql", "TestTransaction")
	if err != nil {
		t.Fatalf("sql.Open : Error : %s\n", err)
	}
	defer db.Close()

	init := []string{
		`CREATE TABLE account (id INT, email TEXT)`,
		`INSERT INTO account (id, email) VALUES (1, 'foo@bar.com')`,
		`INSERT INTO account (id, email) VALUES (2, 'bar@bar.com')`,
		`CREATE TABLE champion (user_id INT, name TEXT)`,
		`INSERT INTO champion (user_id, name) VALUES (1, 'zed')`,
		`INSERT INTO champion (user_id, name) VALUES (2, 'lulu')`,
		`INSERT INTO champion (user_id, name) VALUES (1, 'thresh')`,
		`INSERT INTO champion (user_id, name) VALUES (1, 'lux')`,
	}
	for _, q := range init {
		_, err = db.Exec(q)
		if err != nil {
			t.Fatalf("sql.Exec: Error: %s\n", err)
		}
	}

	db.SetMaxOpenConns(10)
	var wg sync.WaitGroup

	for i := 0; i < 15; i++ {
		wg.Add(1)
		go execTestTransactionQuery(t, db, &wg)
	}

	wg.Wait()
}

func execTestTransactionQuery(t *testing.T, db *sql.DB, wg *sync.WaitGroup) {

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Cannot create tx: %s", err)
	}

	// Select count
	var count int
	err = tx.QueryRow("SELECT COUNT(user_id) FROM champion WHERE user_id = 1").Scan(&count)
	if err != nil {
		t.Fatalf("cannot query row in tx: %s\n", err)
	}
	if count != 3 {
		t.Fatalf("expected COUNT(user_id)=3 row, got %d", count)
	}

	err = tx.Commit()
	if err != nil {
		t.Fatalf("cannot commit tx: %s", err)
	}

	wg.Done()
}

func TestTransactionRollback(t *testing.T) {
	log.SetLevel(log.WarningLevel)
	defer log.SetLevel(log.ErrorLevel)

	db, err := sql.Open("ramsql", "TestTransactionRollback")
	if err != nil {
		t.Fatalf("sql.Open : Error : %s\n", err)
	}
	defer db.Close()

	init := []string{
		`CREATE TABLE account (id INT, email TEXT)`,
		`INSERT INTO account (id, email) VALUES (1, 'foo@bar.com')`,
		`INSERT INTO account (id, email) VALUES (2, 'bar@bar.com')`,
		`CREATE TABLE champion (user_id INT, name TEXT)`,
		`INSERT INTO champion (user_id, name) VALUES (1, 'zed')`,
		`INSERT INTO champion (user_id, name) VALUES (2, 'lulu')`,
		`INSERT INTO champion (user_id, name) VALUES (1, 'thresh')`,
		`INSERT INTO champion (user_id, name) VALUES (1, 'lux')`,
	}
	for _, q := range init {
		_, err = db.Exec(q)
		if err != nil {
			t.Fatalf("sql.Exec: Error: %s\n", err)
		}
	}

	var count int
	err = db.QueryRow("SELECT COUNT(user_id) FROM champion WHERE user_id = 1").Scan(&count)
	if err != nil {
		t.Fatalf("cannot query row in tx: %s\n", err)
	}
	if count != 3 {
		t.Fatalf("expected COUNT(user_id)=3 row, got %d", count)
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("cannot begin transaction: %s", err)
	}

	_, err = tx.Exec(`INSERT INTO champion (user_id, name) VALUES (1, 'new-champ')`)
	if err != nil {
		t.Fatalf("cannot insert within transaction: %s", err)
	}

	err = tx.QueryRow("SELECT COUNT(*) FROM champion WHERE user_id = 1").Scan(&count)
	if err != nil {
		t.Fatalf("cannot query row in tx: %s\n", err)
	}
	if count != 4 {
		t.Fatalf("expected COUNT(user_id)=4 row within tx, got %d", count)
	}

	err = tx.Rollback()
	if err != nil {
		t.Fatalf("cannot rollback transaction: %s", err)
	}

	err = db.QueryRow("SELECT COUNT(user_id) FROM champion WHERE user_id = 1").Scan(&count)
	if err != nil {
		t.Fatalf("cannot query row in tx: %s\n", err)
	}
	if count != 3 {
		t.Fatalf("expected COUNT(user_id)=3 row, got %d", count)
	}
}

func TestCheckAttributes(t *testing.T) {

	db, err := sql.Open("ramsql", "TestCheckAttribute")
	if err != nil {
		t.Fatalf("sql.Open : Error : %s\n", err)
	}
	defer db.Close()

	init := []string{
		`CREATE TABLE account (id INT, email TEXT)`,
		`INSERT INTO account (id, email) VALUES (1, 'foo@bar.com')`,
		`INSERT INTO account (id, email) VALUES (2, 'bar@bar.com')`,
	}
	for _, q := range init {
		_, err = db.Exec(q)
		if err != nil {
			t.Fatalf("sql.Exec: Error: %s\n", err)
		}
	}

	query := `INSERT INTO account(id, nonexisting_attribute) VALUES (1, foo)`
	_, err = db.Exec(query)
	if err == nil {
		t.Errorf("expected an error trying to insert non existing attribute")
	}

	query = `SELECT * FROM account WHERE nonexisting_attribute = 2`
	_, err = db.Query(query)
	if err == nil {
		t.Errorf("expected an error trying to make a comparison with a non existing attribute")
	}

	query = `SELECT id, nonexisting_attribute FROM account WHERE id = 2`
	rows, err := db.Query(query)
	if err == nil {
		t.Errorf("expected an error trying to select a non existing attribute")
	}
	_ = rows
}
