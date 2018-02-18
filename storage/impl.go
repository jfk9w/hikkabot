package storage

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dgraph-io/badger"
	"github.com/jfk9w/hikkabot/webm"
	"github.com/pkg/errors"
)

const (
	prefixActive   = "active"
	prefixInactive = "inactive"
	prefixWebm     = "webm"
	path0          = "!"
	path1          = ":"
	path2          = "/"
)

type impl struct {
	config Config
	db     *badger.DB
}

func New(config Config, db *badger.DB) *impl {
	return &impl{config, db}
}

func activeKey(acc AccountID, thr ThreadID) []byte {
	return []byte(
		prefixActive + path0 +
			acc + path1 +
			thr,
	)
}

func inactiveKey(active []byte) []byte {
	ts := strings.Split(string(active), path0)
	return []byte(prefixInactive + path0 + ts[1])
}

func (s *impl) DumpState() (State, error) {
	state := make(map[AccountID]map[ThreadID]int)
	if err := s.db.View(func(tx *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := tx.NewIterator(opts)
		defer it.Close()
		prefix := []byte(prefixActive)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			k := item.Key()
			v, err := item.Value()
			if err != nil {
				return err
			}

			t0 := strings.Split(string(k), path0)
			ts := strings.Split(t0[1], path1)
			acc, thr := ts[0], ts[1]
			offset, err := strconv.Atoi(string(v))
			if err != nil {
				return err
			}

			if _, ok := state[acc]; !ok {
				state[acc] = make(map[ThreadID]int)
			}

			state[acc][thr] = offset
		}

		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "select all failed")
	}

	return state, nil
}

func (s *impl) Resume(acc AccountID, thr ThreadID) error {
	_, offset := ReadThreadID(thr)
	_, err := strconv.Atoi(offset)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("[%s] invalid thread ID: %s", acc, thr))
	}

	k := activeKey(acc, thr)
	return errors.Wrap(
		s.db.Update(func(tx *badger.Txn) error {
			_, err := tx.Get(k)
			if err == badger.ErrKeyNotFound {
				var v []byte
				susp := inactiveKey(k)
				item, err := tx.Get(susp)
				if err == nil {
					v, err = item.Value()
					if err != nil {
						return err
					}

					err = tx.Delete(susp)
					if err != nil {
						return err
					}
				} else if err == badger.ErrKeyNotFound {
					v = []byte(offset)
				} else {
					return err
				}

				return tx.Set(k, v)
			}

			return err
		}),
		fmt.Sprintf("[%s] resume failed for %s", acc, thr),
	)
}

func (s *impl) Suspend(acc AccountID, thr ThreadID) error {
	k := activeKey(acc, thr)
	return errors.Wrap(
		s.db.Update(func(tx *badger.Txn) error {
			item, err := tx.Get(k)
			if err != nil && err != badger.ErrKeyNotFound {
				return err
			}

			v, err := item.Value()
			if err != nil {
				return err
			}

			if err := tx.SetWithTTL(inactiveKey(k), v,
				s.config.InactiveTTL); err != nil {
				return err
			}

			return tx.Delete(k)
		}),
		fmt.Sprintf("[%s] failed to suspend %s", acc, thr),
	)
}

func (s *impl) SuspendAll(acc AccountID) error {
	return errors.Wrap(
		s.db.Update(func(tx *badger.Txn) error {
			opts := badger.DefaultIteratorOptions
			it := tx.NewIterator(opts)

			keys := make([][]byte, 0)
			prefix := []byte(prefixActive + path0 + acc)
			for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
				item := it.Item()
				k := item.Key()
				v, err := item.Value()
				if err != nil {
					it.Close()
					return err
				}

				if err := tx.SetWithTTL(inactiveKey(k), v,
					s.config.InactiveTTL); err != nil {
					it.Close()
					return err
				}

				keys = append(keys, k)
			}

			it.Close()
			for _, k := range keys {
				if err := tx.Delete(k); err != nil {
					return err
				}
			}

			return nil
		}),
		fmt.Sprintf("[%s] failed to suspend all", acc),
	)
}

func (s *impl) IsActive(acc AccountID, thr AccountID) (bool, error) {
	k := activeKey(acc, thr)
	var r bool
	if err := s.db.View(func(tx *badger.Txn) error {
		_, err := tx.Get(k)
		if err == nil {
			r = true
			return nil
		} else if err == badger.ErrKeyNotFound {
			r = false
			return nil
		}

		return err
	}); err != nil {
		return false, errors.Wrap(err, fmt.Sprintf("[%s] status check failed for %s", acc, thr))
	}

	return r, nil
}

func (s *impl) Update(acc AccountID, thr ThreadID, offset int) error {
	k := activeKey(acc, thr)
	return errors.Wrap(
		s.db.Update(func(tx *badger.Txn) error {
			_, err := tx.Get(k)
			if err == badger.ErrKeyNotFound {
				return nil
			} else if err != nil {
				return err
			}

			v := []byte(strconv.Itoa(offset))
			return tx.Set(k, v)
		}),
		fmt.Sprintf("[%s] offset update failed for %s (%d)", acc, thr, offset),
	)
}

func webmKey(url string) []byte {
	return []byte(prefixWebm + path0 + url)
}

func (s *impl) GetVideo(url string) (string, error) {
	var r string
	if err := s.db.View(func(tx *badger.Txn) error {
		item, err := tx.Get(webmKey(url))
		switch err {
		case badger.ErrKeyNotFound:
			r = webm.NotFound
		case nil:
			v, err := item.Value()
			if err == nil {
				r = string(v)
			} else {
				r = webm.Marked
			}
		default:
			return err
		}

		return nil
	}); err != nil {
		return "", errors.Wrap(err, fmt.Sprintf(
			"get video failed for %s", url))
	}

	return r, nil
}

func (s *impl) CompareAndSwapVideo(url string, prev string, curr string) bool {
	r := false
	key := webmKey(url)
	s.db.Update(func(tx *badger.Txn) error {
		item, err := tx.Get(key)
		if err != nil {
			return nil
		}

		v, err := item.Value()
		if err != nil {
			return nil
		}

		if prev == string(v) {
			tx.SetWithTTL(key, []byte(curr), s.config.VideoTTL)
			r = true
		}

		return nil
	})

	return r
}
