// Package internal provides
package internal

import (
	"context"
	"fmt"
	"testing"

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

	err := adapter.Transaction(ctx, func(ctx context.Context, a *Adapter) error {
		var user User

		err := a.conn.Model(User{}).Where("name = ?", "name_11").First(&user).Error
		if err != nil {
			return err
		}

		fmt.Printf("user: %v\n", user)

		err = a.conn.Model(User{ID: user.ID}).Update("name", "name_12").Error
		if err != nil {
			return err
		}

		return nil
	})
	require.NoError(t, err)
}

func TestChannelRange(t *testing.T) {
	var req chan struct{}
	var reqKey uint64

	requests := make(map[uint64]chan struct{})
	for i := 0; i < 3; i++ {
		requests[uint64(i+1)] = make(chan struct{})
	}

	for reqKey, req = range requests {
		break
	}

	fmt.Printf("reqKey: %v\n", reqKey)
	fmt.Printf("req: %v\n", req)
}
