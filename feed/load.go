package feed

import (
	"html"
	"sync"

	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/gox/syncx"
	"github.com/jfk9w-go/hikkabot/text"
	"github.com/jfk9w-go/httpx"
	"github.com/jfk9w-go/telegram"
)

type Load interface {
	HasNext() bool
	Next(chan<- Event)
}

type DummyLoad struct{}

func (_ *DummyLoad) HasNext() bool {
	return false
}

func (_ *DummyLoad) Next(events chan<- Event) {
	close(events)
}

type DvachLoad struct {
	Dvach
	Aconvert
	*DvachMeta
	Posts []*dvach.Post
	Index int
}

func (load *DvachLoad) HasNext() bool {
	return load.Index < len(load.Posts)
}

func (load *DvachLoad) Next(events chan<- Event) {
	var (
		post  = load.Posts[load.Index]
		files = syncx.NewMap()
		group sync.WaitGroup
	)

	load.Index += 1

	group.Add(len(post.Files))
	for _, dfile := range post.Files {
		go func(dfile *dvach.File) {
			var (
				url  = dfile.URL()
				file = new(httpx.File)
				err  = load.Dvach.Get(url, nil, file)
			)

			if dfile.Type == dvach.Webm {
				url, err = load.Aconvert.Convert(file)
				file.Delete()
				if err != nil {
					goto wrap
				}

				err = load.Aconvert.Get(url, nil, file)
			}

		wrap:
			if err == nil {
				files.Put(dfile.URL(), file)
			}

			group.Done()
		}(dfile)
	}

	if load.Mode != MediaDvachMode {
		var parts = text.FormatPost(text.Post{post, load.Title})
		for _, part := range parts {
			events <- &TextItem{part}
		}
	}

	group.Wait()
	for _, dfile := range post.Files {
		var link = `<a href="` + html.EscapeString(dfile.URL()) + `">[A]</a>`
		if load.Mode == MediaDvachMode {
			link += "\n" + load.Title
		}

		if any, ok := files.Get(dfile.URL()); ok {
			var file = any.(*httpx.File)
			switch dfile.Type {
			case dvach.Gif, dvach.Webm, dvach.Mp4:
				if file.Size > telegram.MaxVideoSize {
					events <- &TextItem{link}
				} else {
					events <- &VideoItem{file, link}
				}

			default:
				if file.Size > telegram.MaxPhotoSize {
					events <- &TextItem{link}
				} else {
					events <- &ImageItem{file, link}
				}
			}
		} else {
			events <- &TextItem{link}
		}
	}

	events <- &End{post.NumString}
	close(events)
}
