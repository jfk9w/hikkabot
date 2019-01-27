package httpx

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"testing"

	"github.com/jfk9w-go/hikkabot/common/gox/fsx"
	"github.com/jfk9w-go/hikkabot/common/gox/testx"
)

func serve(t *testing.T) int {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	go http.Serve(listener, http.FileServer(http.Dir("./testdata")))
	return listener.Addr().(*net.TCPAddr).Port
}

func TestFileOutput(t *testing.T) {
	port := serve(t)

	assert := testx.Assert(t)
	http := Configure(&Config{
		Transport: &TransportConfig{
			Log: "httpx-test",
		},
	})

	dir := fsx.Join(os.TempDir(), "httpx")
	path := fsx.Join(dir, "4uFqvdPP9u4.jpg")

	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}

	defer os.RemoveAll(dir)

	if err := fsx.EnsureParent(path); err != nil {
		t.Fatal(err)
	}

	output := &File{
		Path: path,
	}

	err := http.Get(
		fmt.Sprintf("http://localhost:%d/4uFqvdPP9u4.jpg", port),
		nil, output,
	)

	if err != nil {
		t.Fatal(err)
	}

	stat, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	assert.EqualsInt64(130140, stat.Size())
}
