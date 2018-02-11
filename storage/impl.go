package storage

import (
	"strconv"
	"strings"

	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
)

const (
	prefixOn  = "on"
	prefixOff = "off"
	path0     = "$"
	path1     = "#"
)

type impl struct {
	config Config
	db     *badger.DB
}

func on(acc AccountID, thr ThreadID) []byte {
	return []byte(
		prefixOn + path0 +
			acc.Key() + path1 +
			NewThreadKey(thr),
	)
}

func off(on []byte) []byte {
	ts := strings.Split(string(on), path0)
	return []byte(prefixOff + path0 + ts[1])
}

func (s *impl) SelectAll() (State, error) {
	state := make(map[AccountKey]map[ThreadKey]int)
	if err := s.db.View(func(tx *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := tx.NewIterator(opts)
		defer it.Close()
		prefix := []byte(prefixOn)
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
				state[acc] = make(map[ThreadKey]int)
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
	offset := thr[1]
	_, err := strconv.Atoi(offset)
	if err != nil {
		return errors.Wrap(err, "invalid thread ID")
	}

	k := on(acc, thr)
	return errors.Wrap(
		s.db.Update(func(tx *badger.Txn) error {
			_, err := tx.Get(k)
			if err == badger.ErrKeyNotFound {
				var v []byte
				susp, err := tx.Get(off(k))
				if err == nil {
					v, err = susp.Value()
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
		"resume failed",
	)
}

func (s *impl) Suspend(acc AccountID, thr ThreadID) error {
	k := on(acc, thr)
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

			if err := tx.SetWithTTL(off(k), v,
				s.config.SubscriptionTTL); err != nil {
				return err
			}

			return tx.Delete(k)
		}),
		"suspend failed",
	)
}

func (s *impl) SuspendAll(acc AccountID) error {
	return errors.Wrap(
		s.db.Update(func(tx *badger.Txn) error {
			opts := badger.DefaultIteratorOptions
			it := tx.NewIterator(opts)

			keys := make([][]byte, 0)
			prefix := []byte(acc.Key())
			for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
				item := it.Item()
				k := item.Key()
				v, err := item.Value()
				if err != nil {
					it.Close()
					return err
				}

				if err := tx.SetWithTTL(off(k), v,
					s.config.SubscriptionTTL); err != nil {
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
		"suspend all failed",
	)
}

func (s *impl) IsActive(acc AccountID, thr ThreadID) (bool, error) {
	k := on(acc, thr)
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
		return false, errors.Wrap(err, "is active failed")
	}

	return r, nil
}

func (s *impl) Update(acc AccountID, thr ThreadID, offset int) error {
	k := on(acc, thr)
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
		"update failed",
	)
}
