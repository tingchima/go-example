// Package internal provides
package internal

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

var repository *Repository

func init() {
	repository = MustNewRepository()
}

func TestReadWriteSplitting_Basic(t *testing.T) {
	db := repository.conn

	dbCfg := db.Config
	fmt.Printf("db config: %v\n", dbCfg)

	connPool := dbCfg.ConnPool
	fmt.Printf("db conn pool: %v\n", connPool)

	// read db
	{
		var users []User
		err := db.Model(User{}).Find(&users).Error
		require.NoError(t, err)

		fmt.Printf("before updated users: %v\n", users)
	}

	// write db
	{
		err := db.Model(User{ID: 10}).Update("name", "name_13").Error
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

func TestReadWriteSplitting_Transaction(t *testing.T) {
	ctx := context.Background()
	userID := 1

	fmt.Println("start tx...")
	err := repository.Transaction(ctx, func(ctx context.Context, tx *Repository) error {
		var user User
		fmt.Println("start find user...")
		err := tx.conn.Model(User{}).Where("id = ?", userID).Find(&user).Error
		if err != nil {
			return err
		}

		fmt.Printf("user before updated: %v\n", user)

		err = tx.conn.Model(User{ID: user.ID}).Update("name", uuid.New().String()).Error
		if err != nil {
			return err
		}

		err = tx.conn.Model(User{}).Where("id = ?", user.ID).Find(&user).Error
		if err != nil {
			return err
		}

		fmt.Printf("user after updated: %v\n", user)

		return nil
	})
	require.NoError(t, err)

	var user User
	err = repository.conn.WithContext(ctx).Where("id = ?", userID).First(&user).Error
	require.NoError(t, err)

	fmt.Printf("nontx-user after updated: %v\n", user)
}

// Test .
func TestReadWriteSplitting_GormTransaction(t *testing.T) {
	userID := 1

	err := repository.conn.Transaction(func(tx *gorm.DB) error {
		var user User
		err := tx.Model(User{}).Where("id = ?", userID).Find(&user).Error
		if err != nil {
			return err
		}

		fmt.Printf("user before updated: %v\n", user)

		err = tx.Model(User{ID: user.ID}).Update("name", uuid.New().String()).Error
		if err != nil {
			return err
		}

		err = tx.Model(User{}).Where("id = ?", user.ID).Find(&user).Error
		if err != nil {
			return err
		}

		fmt.Printf("user after updated: %v\n", user)

		return nil
	})
	require.NoError(t, err)

	var user User
	var ctx = context.Background()
	err = repository.conn.WithContext(ctx).Where("id = ?", userID).First(&user).Error
	require.NoError(t, err)

	fmt.Printf("nontx-user after updated: %v\n", user)
}
