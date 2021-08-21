package gorm

import (
	"fmt"
	"testing"

	"github.com/ory/dockertest"
	"gorm.io/gorm"
)

type TestDatabase struct {
	*gorm.DB
	Close func() error
}

func NewTestDatabase(t *testing.T) *TestDatabase {
	pool, err := dockertest.NewPool("")
	if err != nil {
		t.Fatal(err)
	}

	container, err := pool.Run("postgres", "latest", []string{"POSTGRES_PASSWORD=pwd"})
	if err != nil {
		t.Fatal(err)
	}

	var db *gorm.DB
	dsn := fmt.Sprintf("postgresql://postgres:pwd@localhost:%s/postgres", container.GetPort("5432/tcp"))
	if err := pool.Retry(func() error {
		db, err = NewPostgres(dsn)
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
