package blob

import (
	"context"
	"math/rand"
	"os"
	"time"

	"github.com/jfk9w/hikkabot/core/feed"

	"github.com/jfk9w-go/flu"
	"github.com/sirupsen/logrus"
)

type FileStorageConfig struct {
}

type FileStorage struct {
	Directory string
	TTL       time.Duration
	files     map[flu.File]time.Time
	work      flu.WaitGroup
	cancel    func()
	mu        flu.Mutex
}

func (s *FileStorage) Init() error {
	s.Remove()
	if err := os.MkdirAll(s.Directory, 0755); err != nil {
		return err
	}

	s.files = make(map[flu.File]time.Time)
	return nil
}

func (s *FileStorage) ScheduleMaintenance(ctx context.Context, every time.Duration) {
	if s.cancel != nil {
		return
	}

	s.cancel = s.work.Go(ctx, func(ctx context.Context) {
		ticker := time.NewTicker(every)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				s.RemoveStaleFiles(now)
			}
		}
	})
}

func (s *FileStorage) Close() error {
	if s.cancel != nil {
		s.cancel()
		s.work.Wait()
	}

	return nil
}

var (
	symbols  = []rune("abcdefghijklmonpqrstuvwxyz0123456789")
	idLength = 16
)

func (s *FileStorage) newID() string {
	id := make([]rune, idLength)
	for i := 0; i < idLength; i++ {
		id[i] = symbols[rand.Intn(len(symbols))]
	}

	return string(id)
}

func (s *FileStorage) Alloc(now time.Time) (feed.Blob, error) {
	defer s.mu.Lock().Unlock()
	file := flu.File(s.Directory + "/" + s.newID())
	s.files[file] = now
	return file, nil
}

func (s *FileStorage) Remove() {
	_ = os.RemoveAll(s.Directory)
}

func (s *FileStorage) RemoveStaleFiles(now time.Time) {
	defer s.mu.Lock().Unlock()
	count := 0
	for file, createdAt := range s.files {
		if now.Sub(createdAt) > s.TTL {
			if err := os.RemoveAll(file.Path()); err != nil {
				logrus.WithField("file", file).
					Warnf("failed to remove blob file: %s", err)
				continue
			}

			delete(s.files, file)
			count++
		}
	}

	logrus.Infof("removed %d stale files", count)
}
