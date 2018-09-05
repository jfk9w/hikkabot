package feed

import (
	"html"
	"sync"

	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/gox/syncx"
	"github.com/jfk9w-go/hikkabot/text"
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
		var parts = text.FormatPost(text.Post{post, load.Title})
		for _, part := range parts {
			events <- &TextItem{part}
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

	events <- &End{post.Num}
	close(events)
}

type RedLoad struct {
	Red
	Data  []red.ThingData
	Index int
}

var AllowedRedDomains = []string{
	"i.redd.it", "i.imgur.com", "imgur.com",
}

func (load *RedLoad) IsAllowed(data red.ThingData) bool {
	for _, allowed := range AllowedRedDomains {
		if data.Domain == allowed {
			return true
		}
	}

	return false
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

		if err == nil {
			events <- &ImageItem{file, caption}
		} else {
			events <- &TextItem{caption}
		}
	} else {
		log.Warnf("Unsupported domain: %s, url: %s", data.Domain, data.URL)
	}

	events <- &End{int(data.CreatedUTC)}
	close(events)
}
