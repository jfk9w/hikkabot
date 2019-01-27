package dvach

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestApi_Catalog(t *testing.T) {
	var _, err = Configure(Config{}).Catalog("b")
	if err != nil {
		t.Fatal(err)
	}
}

func TestApi_Thread(t *testing.T) {
	var url = "https://2ch.hk/abu/res/42375.html"
	var ref, err = ParseUrl(url)
	if err != nil {
		t.Fatal(err)
	}

	var c = Configure(Config{})
	ps, err := c.Posts(ref, 0)
	if err != nil {
		t.Fatal(err)
	}

	if ps[0].Num != ref.Num {
		t.Fatalf("first post num (%s) differs from thread num (%s)", ps[0].NumString, ref.NumString)
	}

	var offset = 49954
	ps, err = c.Posts(ref, offset)
	if err != nil {
		t.Fatal(err)
	}

	if ps[0].Num != offset {
		t.Fatalf("first post num (%s) differs from offset (%d)", ps[0].NumString, offset)
	}
}

func TestApi_Error(t *testing.T) {
	var url = "https://2ch.hk/abu/res/42376.html"
	var ref, _ = ParseUrl(url)
	_, err := Configure(Config{}).Posts(ref, 0)
	var e = err.(*Error)
	spew.Dump(e)
}
