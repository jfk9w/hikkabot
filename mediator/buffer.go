package mediator

import (
	"os"

	"github.com/jfk9w-go/flu"
	"github.com/otiai10/gosseract/v2"
)

type Buffer interface {
	flu.ReadWritable
	Cleanup()
	setOCR(*gosseract.Client)
}

type memoryBuffer struct {
	flu.Buffer
}

func (mb memoryBuffer) Cleanup() {

}

func (mb memoryBuffer) setOCR(client *gosseract.Client) {
	client.SetImageFromBytes(mb.Bytes())
}

type fileBuffer struct {
	flu.File
}

func (fb fileBuffer) Cleanup() {
	os.RemoveAll(fb.File.Path())
}

func (fb fileBuffer) setOCR(client *gosseract.Client) {
	client.SetImage(fb.Path())
}
