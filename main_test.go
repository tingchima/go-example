package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

// User .
type User struct {
	ID   int64
	Name string
}

// TableName .
func (u *User) TableName() string {
	return "users"
}

// TestReadWriteSplitting .
func TestReadWriteSplitting(t *testing.T) {
	dsn := "%s:%s@(%s:%s)/%s?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci&loc=UTC"

	dsn1 := fmt.Sprintf(dsn, "root", "1234", "127.0.0.1", "33061", "go_example")
	dsn2 := fmt.Sprintf(dsn, "root", "1234", "127.0.0.1", "33062", "go_example")

	fmt.Printf("dsn1: %v\n", dsn1)
	fmt.Printf("dsn2: %v\n", dsn2)

	// DB's default connection
	db, err := gorm.Open(mysql.Open(dsn1), &gorm.Config{})
	require.NoError(t, err)

	// use `db1` as sources (DB's default connection)
	// use `db2`, `db3` as replicas
	err = db.Use(
		dbresolver.Register(dbresolver.Config{
			Sources: []gorm.Dialector{},
			Replicas: []gorm.Dialector{
				mysql.Open(dsn2),
				// add more dsn...
			},
			Policy: dbresolver.RandomPolicy{},
		}).
			SetConnMaxIdleTime(time.Hour).
			SetConnMaxLifetime(24 * time.Hour).
			SetMaxIdleConns(100).
			SetMaxOpenConns(200),
	)
	require.NoError(t, err)

	// read db
	{
		var users []User
		err := db.Model(User{}).Find(&users).Error
		require.NoError(t, err)

		fmt.Printf("before updated users: %v\n", users)
	}

	// write db
	{
		err = db.Model(User{ID: 10}).Update("name", "name_10").Error
		require.NoError(t, err)
	}

	{
		var users []User
		err := db.Model(User{}).Find(&users).Error
		require.NoError(t, err)

		fmt.Printf("after updated users: %v\n", users)
	}

	// manual
	// db.Clauses(dbresolver.Write)
}
