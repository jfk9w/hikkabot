package common

import (
	"fmt"
	"testing"

	"github.com/jfk9w-go/flu/gormf"
	"github.com/ory/dockertest"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type TestDatabase struct {
	*gorm.DB
	Close func() error
}

func NewTestPostgres(t *testing.T) *TestDatabase {
	pool, err := dockertest.NewPool("")
	if err != nil {
		t.Fatal(err)
	}

	container, err := pool.Run("postgres", "latest", []string{"POSTGRES_PASSWORD=pwd"})
	if err != nil {
		t.Fatal(err)
	}

	var db *gorm.DB
	dsn := fmt.Sprintf("postgresql://postgres:pwd@%s/postgres", container.GetHostPort("5432/tcp"))
	if err := pool.Retry(func() error {
		db, err = gorm.Open(postgres.Open(dsn), gormf.DefaultConfig)
		return err
	}); err != nil {
		t.Fatal(err)
	}

	closer := func() error {
		db, err := db.DB()
		if err != nil {
			return err
		}

		if err := db.Close(); err != nil {
			return err
		}

		return container.Close()
	}

	return &TestDatabase{db, closer}
}
