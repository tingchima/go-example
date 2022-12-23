// Package internal provides
package internal

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

var adapter *Adapter

func init() {
	adapter = MustNewAdapter()
}

func TestReadWriteSplitting_Basic(t *testing.T) {
	db := adapter.conn

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

	err := adapter.Transaction(ctx, func(ctx context.Context, tx *Adapter) error {
		var user User

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
	err = adapter.conn.WithContext(ctx).Where("id = ?", userID).First(&user).Error
	require.NoError(t, err)

	fmt.Printf("nontx-user after updated: %v\n", user)
}
