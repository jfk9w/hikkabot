package feed

import (
	"html"
	"sync"

	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/gox/syncx"
	"github.com/jfk9w-go/hikkabot/content"
	"github.com/jfk9w-go/httpx"
	"github.com/jfk9w-go/red"
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

func A(link string) string {
	return `<a href="` + html.EscapeString(link) + `">[A]</a>`
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
		var parts = content.FormatDvachPost(post, load.Title)
		for _, part := range parts {
			events <- &TextItem{Text: part}
		}
	}

	group.Wait()
	for _, dfile := range post.Files {
		var link = A(dfile.URL())
		if load.Mode == MediaDvachMode {
			link += "\n" + load.Title
		}

		if any, ok := files.Get(dfile.URL()); ok {
			var file = any.(*httpx.File)
			switch dfile.Type {
			case dvach.Gif, dvach.Webm, dvach.Mp4:
				if file.Size > telegram.MaxVideoSize {
					events <- &TextItem{Text: link}
				} else {
					events <- &VideoItem{file, link}
				}

			default:
				if file.Size > telegram.MaxPhotoSize {
					events <- &TextItem{Text: link}
				} else {
					events <- &ImageItem{file, link}
				}
			}
		} else {
			events <- &TextItem{Text: link}
		}
	}

	events <- &End{post.Num}
	close(events)
}

type DvachWatchLoad struct {
	Result []string
	Offset []Offset
	Index  int
}

func (load *DvachWatchLoad) Get(i int) (string, Offset) {
	return load.Result[i], load.Offset[i]
}

func (load *DvachWatchLoad) HasNext() bool {
	return load.Index < len(load.Offset)
}

func (load *DvachWatchLoad) Next(events chan<- Event) {
	var result, offset = load.Get(load.Index)
	load.Index += 1

	events <- &TextItem{Text: result, DisableWebPagePreview: true}
	events <- &End{offset}
	close(events)
}

type RedLoad struct {
	Red
	Data  []red.ThingData
	Index int
}

var AllowedRedDomains = map[string]struct{}{
	"i.redd.it":   {},
	"i.imgur.com": {},
	"imgur.com":   {},
}

func (load *RedLoad) IsAllowed(data red.ThingData) bool {
	var _, ok = AllowedRedDomains[data.Domain]
	return ok
}

func (load *RedLoad) HasNext() bool {
	return load.Index < len(load.Data)
}

func (load *RedLoad) Next(events chan<- Event) {
	var data = load.Data[load.Index]
	load.Index += 1

	var caption = "#" + data.Subreddit + "\n" + data.Title + "\n" + A(data.URL)
	if load.IsAllowed(data) {
		var (
			file = new(httpx.File)
			err  = load.Red.Get(data.URL, nil, file)
		)

		if err == nil && file.Size <= telegram.MaxPhotoSize {
			events <- &ImageItem{file, caption}
		} else {
			if file != nil {
				file.Delete()
			}

			events <- &TextItem{Text: caption}
		}
	} else {
		log.Warnf("Unsupported domain: %s, url: %s", data.Domain, data.URL)
	}

	events <- &End{int(data.CreatedUTC)}
	close(events)
}
