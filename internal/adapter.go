// Package internal provides
package internal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math/rand"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

/*
R/W Splitting
TODO:
1. 確認基本使用: dbresolver 會根據 R or W 方法，自行切換 master(W) or slave(R) 連線
2. 確認transaction: 需要讓 dbresolver 知道要開啟 R or W 連線當作 transaction 起始點，如果沒有指定會使用 master 當作連線起始點
3. 在 R/W Splitting 下，連線池是怎麼運作的，會拿錯連線來使用嗎？
4. R/W Splitting 延遲問題(BinLog): 要先去理解 mysql 的實作方式 (orz
5. gorm dbresolver 壓測
6. 讀寫分離模式與緩存服務不一致問題:
99. others...
*/

// MustNewRepository .
func MustNewRepository() *Repository {
	adapter, err := NewRepository()
	if err != nil {
		log.Fatalf("new adapter fail: %s", err.Error())
	}
	return adapter
}

// CustomPolicy .
type CustomPolicy struct {
}

// Resolve .
func (CustomPolicy) Resolve(connPools []gorm.ConnPool) gorm.ConnPool {
	fmt.Printf("connection pool resolve method...\n")
	return connPools[rand.Intn(len(connPools))]
}

// Config .
type Config struct {
	Sources            []Conn
	Replicas           []Conn
	ConnMaxIdleTimeSec int
	ConnMaxLifetimeSec int
	MaxIdleConns       int
	MaxOpenConns       int
}

// Conn .
type Conn struct {
	Username     string
	Password     string
	Host         string
	Port         int
	DatabaseName string
}

var (
	mysqlDSNTemplate = "%s:%s@(%s:%s)/%s?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci&loc=UTC"

	writeURL = fmt.Sprintf(mysqlDSNTemplate, "root", "1234", "127.0.0.1", "33061", "go_example")

	readURL1 = fmt.Sprintf(mysqlDSNTemplate, "root", "1234", "127.0.0.1", "33062", "go_example")

	readURL2 = fmt.Sprintf(mysqlDSNTemplate, "root", "1234", "127.0.0.1", "33063", "go_example")
)

// NewRepository .
func NewRepository() (*Repository, error) {
	// newLogger := logger.New(
	// 	log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
	// 	logger.Config{
	// 		// SlowThreshold:             1 * time.Millisecond, // Slow SQL threshold
	// 		// LogLevel:                  logger.Silent, // Log level
	// 		IgnoreRecordNotFoundError: true, // Ignore ErrRecordNotFound error for logger
	// 		Colorful:                  true, // Disable color
	// 	},
	// )

	// db, mock, _ := sqlmock.New()
	// testSuite.dbMock = mock
	// testSuite.db, _ = gorm.Open(postgres.New(postgres.Config{
	// 	DriverName: "postgres",
	// 	Conn:       db,
	// }), &gorm.Config{})

	// DB's default connection
	conn, err := gorm.Open(mysql.Open(writeURL), &gorm.Config{
		SkipDefaultTransaction: true,
		// PrepareStmt:            true,
		// Logger:                 newLogger,
	})
	if err != nil {
		return nil, err
	}

	// Multiple sources and replicas to use...
	// dbresolver.Register(dbresolver.Config{}, "").Register(dbresolver.Config{})

	// use `db1` as sources (DB's default connection)
	// use `db2`, `db3` as replicas
	err = conn.Use(
		// resolver can register multiple configs
		dbresolver.Register(dbresolver.Config{
			Sources: []gorm.Dialector{
				mysql.Open(writeURL),
				// add more source db...
			},
			Replicas: []gorm.Dialector{
				mysql.Open(readURL1),
				mysql.Open(readURL2),
				// add more replica db...
			},
			Policy: dbresolver.RandomPolicy{}, // 可以客製化(參考xorm)
			// TraceResolverMode: true,
		}, "secondary", "users").
			// 會分別對 write / read connection pool 做相同設定值
			SetMaxOpenConns(10).
			SetMaxIdleConns(2),
	)
	if err != nil {
		return nil, err
	}

	// // write
	// sqlDB, err := conn.DB()
	// if err != nil {
	// 	return nil, err
	// }
	// sqlDB.SetMaxOpenConns(10)

	// 這只會對 write(if connection is created with write url) connection pool 設定值
	// sqlDB, err := conn.DB()
	// if err != nil {
	// 	return nil, err
	// }
	// // SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
	// sqlDB.SetMaxIdleConns(10)
	// // SetMaxOpenConns sets the maximum number of open connections to the database.
	// sqlDB.SetMaxOpenConns(100)
	// // SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	// sqlDB.SetConnMaxLifetime(time.Hour)

	// // sharding
	// err = conn.Use(sharding.Register(sharding.Config{
	// 	ShardingKey:         "user_id",
	// 	NumberOfShards:      64,
	// 	PrimaryKeyGenerator: sharding.PKSnowflake,
	// }))
	// if err != nil {
	// 	return nil, err
	// }

	return &Repository{conn}, nil
}

// Repository .
type Repository struct {
	conn *gorm.DB
}

// New .
func New(conn *gorm.DB) *Repository {
	return &Repository{conn: conn}
}

// Begin .
func (d *Repository) Begin(ctx context.Context) *Repository {
	tx := d.conn.Clauses(dbresolver.Write)
	tx = tx.Session(&gorm.Session{Context: ctx})
	tx = tx.Begin(&sql.TxOptions{})
	// tx := d.conn.WithContext(ctx).Begin(&sql.TxOptions{})
	return &Repository{conn: tx}
}

// Commit .
func (d *Repository) Commit() error {
	if d.conn == nil {
		return errors.New("connection is nil")
	}
	return d.conn.Commit().Error
}

// Rollback .
func (d *Repository) Rollback() error {
	if d.conn == nil {
		return errors.New("connection is nil")
	}
	return d.conn.Rollback().Error
}

// Transaction .
func (d *Repository) Transaction(ctx context.Context, fn func(context.Context, *Repository) error) (txErr error) {
	tx := d.Begin(ctx)
	defer func() {
		r := recover()
		if r != nil {
			txErr = errors.New(fmt.Sprint(r))
		}
		if txErr != nil {
			_ = tx.Rollback()
		} else {
			_ = tx.Commit()
		}
	}()
	// Invoke function
	txErr = fn(ctx, tx)
	if txErr != nil {
		return txErr
	}
	return nil
}
