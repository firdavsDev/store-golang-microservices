package migrate

import (
	"context"
	"emperror.dev/errors"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mongodb"
	mongodb2 "github.com/mehdihadeli/store-golang-microservice-sample/pkg/mongodb"
	"go.uber.org/zap"
	"path/filepath"
	"runtime"
)

func (config *MigrationConfig) Migrate(ctx context.Context) error {
	if config.SkipMigration {
		zap.L().Info("database migration skipped")
		return nil
	}

	if config.DBName == "" {
		return errors.New("DBName is required in the config.")
	}

	db, err := mongodb2.NewMongoDB(ctx, &mongodb2.MongoDbConfig{
		Host:     config.Host,
		Port:     config.Port,
		User:     config.User,
		Password: config.Password,
		Database: config.DBName,
		UseAuth:  false,
	})
	if err != nil {
		return err
	}

	driver, err := mongodb.WithInstance(db.MongoClient, &mongodb.Config{DatabaseName: config.DBName, MigrationsCollection: config.VersionTable})
	if err != nil {
		return fmt.Errorf("failed to initialize migrator: %w", err)
	}

	// determine the project's root path
	_, callerPath, _, _ := runtime.Caller(0) // nolint:dogsled

	// look for migrations source starting from project's root dir
	sourceURL := fmt.Sprintf(
		"file://%s/../../%s",
		filepath.ToSlash(filepath.Dir(callerPath)),
		filepath.ToSlash(config.MigrationsDir),
	)

	mig, err := migrate.NewWithDatabaseInstance(sourceURL, config.DBName, driver)
	if err != nil {
		return fmt.Errorf("failed to initialize migrator: %w", err)
	}

	if config.TargetVersion == 0 {
		err = mig.Up()
	} else {
		err = mig.Migrate(config.TargetVersion)
	}

	if err == migrate.ErrNoChange {
		return nil
	}

	zap.L().Info("migration finished")
	if err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}
	
	return nil
}
