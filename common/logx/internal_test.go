package logx

import (
	"os"
	"sync"
	"testing"

	"github.com/jfk9w-go/hikkabot/common/gox/fsx"
	"github.com/jfk9w-go/hikkabot/common/gox/testx"
)

func TestInternal(t *testing.T) {
	var (
		assert  = testx.Assert(t)
		tempdir = fsx.Join(os.TempDir(), "logx")

		obj             internal
		custom, d3fault Ptr
		stat            os.FileInfo

		err error
	)

	os.RemoveAll(tempdir)
	defer os.RemoveAll(tempdir)

	if err := os.Setenv("LOGX", "./testdata/config.json"); err != nil {
		t.Fatal(err)
	}

	if err := os.Setenv("TEMPDIR", tempdir); err != nil {
		t.Fatal(err)
	}

	obj = internal{
		config:  config(),
		loggers: new(sync.Map),
	}

	// logger with custom settings
	custom = obj.get("nondefault")

	// this is the same object
	assert.Equals(custom, obj.get("nondefault"))

	custom.Debug("debug")
	custom.Info("info")
	custom.Warn("warn")
	custom.Error("error")

	if stat, err = os.Stat(fsx.Join(tempdir, "nondefault.log")); err != nil {
		t.Fatal(err)
	}

	// only numbers change
	//assert.EqualsInt64(115, stat.Size())
	assert.EqualsInt64(97, stat.Size())

	// logger with default settings
	d3fault = obj.get("d3fault")

	// this is the same object
	assert.Equals(d3fault, obj.get("d3fault"))

	d3fault.Debug("debug")
	d3fault.Info("info")
	d3fault.Warn("warn")
	d3fault.Error("error")

	if stat, err = os.Stat(fsx.Join(tempdir, "default.log")); err != nil {
		t.Fatal(err)
	}

	// only numbers change
	//assert.EqualsInt64(218, stat.Size())
	assert.EqualsInt64(182, stat.Size())
}
