package main

import (
	"sync"

	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/logrus"
)

type Preloader struct {
	dvach.ApiStorage
	log logrus.Logger
}

func Preload(api dvach.ApiStorage) dvach.ApiStorage {
	return Preloader{api, logrus.GetLogger("pre")}
}

func (p Preloader) Posts(ref dvach.Ref, num dvach.Num) (posts []*dvach.Post, err error) {
	posts, err = p.ApiStorage.Posts(ref, num)
	if err == nil {
		wg := sync.WaitGroup{}
		for _, post := range posts {
			for _, file := range post.Files {
				wg.Add(1)
				go func() {
					if err := p.ApiStorage.Download(file); err != nil {
						log.Warningf("Failed to download %s: %s", file.Path, err)
					} else {
						log.Infof("Preloaded %s (%d bytes)", file.Path, file.Size)
					}

					wg.Done()
				}()
			}
		}

		wg.Wait()
	}

	return
}
