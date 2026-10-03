package migration

import (
	"fmt"
	"io/fs"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	// 注册 MySQL driver
	_ "github.com/golang-migrate/migrate/v4/database/mysql"

	// 如果以后需要 PostgreSQL，可以保留
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
)

const defaultMigrationDir = "migrations"

// Up 执行数据库迁移。
func Up(uri string, migrationsFS fs.FS) error {
	return UpFrom(uri, migrationsFS, defaultMigrationDir)
}

// MustUp 执行数据库迁移。
// 迁移失败直接 panic，阻止服务启动。
func MustUp(uri string, migrationsFS fs.FS) {
	if err := Up(uri, migrationsFS); err != nil {
		panic(fmt.Errorf("database migration failed: %w", err))
	}
}

// UpFrom 执行指定目录下的 migration。
func UpFrom(
	uri string,
	migrationsFS fs.FS,
	migrationDir string,
) error {
	uri = strings.TrimSpace(uri)

	if uri == "" {
		return fmt.Errorf("database uri is empty")
	}

	if migrationsFS == nil {
		return fmt.Errorf("migration filesystem is nil")
	}

	migrationDir = strings.TrimSpace(migrationDir)

	if migrationDir == "" {
		return fmt.Errorf("migration directory is empty")
	}

	// 不要使用 net/url.Parse。
	//
	// migrate 的 MySQL URI：
	//
	// mysql://root:root@tcp(127.0.0.1:3306)/sa2
	//
	// 这种 URI 不适合使用 net/url 进行 host/port 校验。

	// 创建 migration source。
	sourceDriver, err := iofs.New(
		migrationsFS,
		migrationDir,
	)
	if err != nil {
		return fmt.Errorf(
			"create migration source: %w",
			err,
		)
	}

	// 注意：
	// 你当前 migrate 版本的 NewWithSourceInstance
	// 签名是：
	//
	// NewWithSourceInstance(
	//     sourceName string,
	//     source.Driver,
	//     databaseURL string,
	// )
	m, err := migrate.NewWithSourceInstance(
		"iofs",
		sourceDriver,
		uri,
	)
	if err != nil {
		_ = sourceDriver.Close()

		return fmt.Errorf(
			"create migrate instance: %w",
			err,
		)
	}

	defer func() {
		// Close 会关闭 source 和 database driver。
		_, _ = m.Close()
	}()

	// 执行 migration。
	if err := m.Up(); err != nil {
		// 数据库已经是最新版本。
		if err == migrate.ErrNoChange {
			return nil
		}

		return fmt.Errorf(
			"run database migration: %w",
			err,
		)
	}

	return nil
}
