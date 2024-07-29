package apfel

import (
	"context"
	"crypto/md5"
	"fmt"
	"strings"
	"sync"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/logf"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type (
	// GormDriver creates a gorm.Dialector from a connection string.
	GormDriver func(conn string) gorm.Dialector
	// GormDrivers is a map of GormDrivers and their names.
	GormDrivers map[string]GormDriver
)

// Gorm is the base gorm.io/gorm application Mixin.
// It serves as configuration template and supported driver registry.
type Gorm[C any] struct {
	// Config will be used for all created gorm.DB instances.
	Config gorm.Config
	// Drivers contains supported drivers and dialectors.
	Drivers GormDrivers
}

func (m *Gorm[C]) String() string {
	return "gorm"
}

func (m *Gorm[C]) Include(ctx context.Context, app MixinApp[C]) error {
	return nil
}

// GormConfig is the configuration for GormDB.
type GormConfig struct {
	DSN    string `yaml:"dsn" doc:"Database connection string." examples:"postgresql://user:pass@host:port/db,sqlite::memory:"`
	Driver string `yaml:"driver" doc:"Database driver to use. Note that concrete driver support depends on the application." examples:"postgres,sqlite"`
}

// GormDB is the "frontend" gorm.io/gorm Mixin.
type GormDB[C any] struct {
	// Config contains configuration for connecting to the database.
	// It is required to fill Config before passing the mixin to MixinApp.Use.
	Config GormConfig
	db     *gorm.DB
	id     string
	once   sync.Once
}

func (m *GormDB[C]) String() string {
	m.once.Do(func() {
		id := fmt.Sprintf("%x", md5.Sum([]byte(m.Config.DSN)))
		if len(id) > 10 {
			id = id[:10]
		}

		m.id = id
	})

	return "gorm.db." + m.id
}

func (m *GormDB[C]) Include(ctx context.Context, app MixinApp[C]) error {
	if m.Config == (GormConfig{}) {
		return errors.New("config must be filled")
	}

	var factory Gorm[C]
	if err := app.Use(ctx, &factory, true); err != nil {
		return err
	}

	dialect := factory.Drivers[m.Config.Driver]
	if dialect == nil {
		return errors.Errorf("unsupported driver [%s], you may want to register it via [%s]", m.Config.Driver, factory.String())
	}

	db, err := gorm.Open(dialect(m.Config.DSN), &factory.Config)
	if err != nil {
		return err
	}

	m.db = db
	if err := app.Manage(ctx, m); err != nil {
		return err
	}

	configData, _ := flu.ToString(flu.PipeInput(JSONViaYAML(m.Config)))
	logf.Get(m).Tracef(ctx, "%s connect ok", strings.Trim(configData, "\n"))
	return nil
}

// DB returns the gorm.DB instance.
func (m *GormDB[C]) DB() *gorm.DB {
	return m.db
}

func (m *GormDB[C]) Close() error {
	db, err := m.db.DB()
	if err != nil {
		return err
	}

	return db.Close()
}
