package service

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dgraph-io/badger"
	"github.com/jfk9w/hikkabot/webm"
	"github.com/pkg/errors"
)

const (
	pTA   = "thrd[a]"
	pTD   = "thrd[d]"
	pW    = "webm"
	path0 = "!"
	path1 = ":"
	path2 = "/"
)

type BadgerStorage struct {
	config Config
	db     *badger.DB
}

func NewBadgerStorage(config Config, opts badger.Options) (*BadgerStorage, error) {
	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}

	return &BadgerStorage{
		config: config,
		db:     db,
	}, nil
}

func kActiveThread(acc AccountID, thr ThreadID) []byte {
	return []byte(
		pTA + path0 +
			acc + path1 +
			thr,
	)
}

func kDeletedThread(active []byte) []byte {
	ts := strings.Split(string(active), path0)
	return []byte(pTD + path0 + ts[1])
}

func (s *BadgerStorage) Load() (State, error) {
	state := make(map[AccountID]map[ThreadID]int)
	if err := s.db.View(func(tx *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := tx.NewIterator(opts)
		defer it.Close()
		prefix := []byte(pTA)
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

func (s *BadgerStorage) Resume(acc AccountID, thr ThreadID) error {
	_, offset := ReadThreadID(thr)
	_, err := strconv.Atoi(offset)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("[%s] invalid thread ID: %s", acc, thr))
	}

	k := kActiveThread(acc, thr)
	return errors.Wrap(
		s.db.Update(func(tx *badger.Txn) error {
			_, err := tx.Get(k)
			if err == badger.ErrKeyNotFound {
				var v []byte
				susp := kDeletedThread(k)
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
					v = []byte("0")
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

func (s *BadgerStorage) Suspend(acc AccountID, thr ThreadID) error {
	k := kActiveThread(acc, thr)
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

			if err := tx.SetWithTTL(kDeletedThread(k), v,
				s.config.ThreadTTL); err != nil {
				return err
			}

			return tx.Delete(k)
		}),
		fmt.Sprintf("[%s] failed to suspend %s", acc, thr),
	)
}

func (s *BadgerStorage) SuspendAll(acc AccountID) error {
	return errors.Wrap(
		s.db.Update(func(tx *badger.Txn) error {
			opts := badger.DefaultIteratorOptions
			it := tx.NewIterator(opts)

			keys := make([][]byte, 0)
			prefix := []byte(pTA + path0 + acc)
			for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
				item := it.Item()
				k := item.Key()
				v, err := item.Value()
				if err != nil {
					it.Close()
					return err
				}

				if err := tx.SetWithTTL(kDeletedThread(k), v,
					s.config.ThreadTTL); err != nil {
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

func (s *BadgerStorage) GetOffset(acc AccountID, thr AccountID) (int, error) {
	k := kActiveThread(acc, thr)
	var r int
	if err := s.db.View(func(tx *badger.Txn) error {
		item, err := tx.Get(k)
		if err == nil {
			var v []byte
			v, err = item.Value()
			if err == nil {
				r, err = strconv.Atoi(string(v))
				if err == nil {
					return nil
				}
			}
		} else if err == badger.ErrKeyNotFound {
			r = -1
			return nil
		}

		return err
	}); err != nil {
		return r, errors.Wrap(err, fmt.Sprintf("[%s] get offset failed for %s", acc, thr))
	}

	return r, nil
}

func (s *BadgerStorage) UpdateOffset(acc AccountID, thr ThreadID,
	offset int) bool {
	r := false
	k := kActiveThread(acc, thr)
	for s.db.Update(func(tx *badger.Txn) error {
		_, err := tx.Get(k)
		if err != nil {
			return err
		}

		v := []byte(strconv.Itoa(offset))

		err = tx.Set(k, v)
		if err != nil {
			return err
		}

		r = true
		return nil
	}) == badger.ErrConflict {
	}

	return r
}

func kWebm(url string) []byte {
	return []byte(pW + path0 + url)
}

func (s *BadgerStorage) GetVideo(url string) (string, error) {
	var r string
	if err := s.db.View(func(tx *badger.Txn) error {
		item, err := tx.Get(kWebm(url))
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

func (s *BadgerStorage) CompareAndSwapVideo(url string, prev string, curr string) bool {
	r := false
	k := kWebm(url)
	s.db.Update(func(tx *badger.Txn) error {
		item, err := tx.Get(k)
		if err != nil {
			return nil
		}

		v, err := item.Value()
		if err != nil {
			return nil
		}

		if prev == string(v) {
			tx.SetWithTTL(k, []byte(curr), s.config.WebmTTL)
			r = true
		}

		return nil
	})

	return r
}

func (s *BadgerStorage) Close() error {
	return s.db.Close()
}
