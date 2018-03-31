package service

import (
	"strconv"
	"strings"

	"github.com/dgraph-io/badger"
	"github.com/jfk9w/hikkabot/webm"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	pTA = "thrd[a]"
	pTD = "thrd[d]"
	pW  = "webm"
	sP  = "\t"
	sT  = "/"
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
		pTA + sP +
			acc + sP +
			thr,
	)
}

func kDeletedThread(active []byte) []byte {
	ts := strings.Split(string(active), sP)
	return []byte(pTD + sP + ts[1] + sP + ts[2])
}

func (s *BadgerStorage) Load() (State, error) {
	state := make(map[AccountID][]ThreadID)
	if err := s.db.View(func(tx *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := tx.NewIterator(opts)
		defer it.Close()
		prefix := []byte(pTA)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			k := item.Key()

			ts := strings.Split(string(k), sP)
			acc, thr := ts[1], ts[2]

			if _, ok := state[acc]; !ok {
				state[acc] = make([]ThreadID, 0)
			}

			state[acc] = append(state[acc], thr)
		}

		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "init failed")
	}

	return state, nil
}

func (s *BadgerStorage) InsertThread(acc AccountID, thr ThreadID) bool {
	r := false
	k := kActiveThread(acc, thr)
	for s.db.Update(func(tx *badger.Txn) error {
		_, err := tx.Get(k)
		if err == nil {
			return nil
		}

		var v []byte
		susp := kDeletedThread(k)
		item, err := tx.Get(susp)
		if err == badger.ErrKeyNotFound {
			v = []byte("0")
		} else {
			v, _ = item.Value()
			tx.Delete(susp)
		}

		r = true
		tx.Set(k, v)

		return nil
	}) == badger.ErrConflict {
	}

	log.WithFields(log.Fields{
		"acc": acc,
		"thr": thr,
		"r":   r,
	}).Debug("BDGR InsertThread")

	return r
}

func (s *BadgerStorage) DeleteThread(acc AccountID, thr ThreadID) {
	k := kActiveThread(acc, thr)
	for s.db.Update(func(tx *badger.Txn) error {
		item, err := tx.Get(k)
		if err == badger.ErrKeyNotFound {
			return nil
		}

		v, _ := item.Value()
		tx.SetWithTTL(kDeletedThread(k), v, s.config.ThreadTTL)
		tx.Delete(k)

		return nil
	}) == badger.ErrConflict {
	}

	log.WithFields(log.Fields{
		"acc": acc,
		"thr": thr,
	}).Debug("BDGR DeleteThread")
}

func (s *BadgerStorage) DeleteAccount(acc AccountID) {
	for s.db.Update(func(tx *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := tx.NewIterator(opts)

		keys := make([][]byte, 0)
		prefix := []byte(pTA + sP + acc)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			k := item.Key()
			v, _ := item.Value()
			tx.SetWithTTL(kDeletedThread(k), v, s.config.ThreadTTL)
			keys = append(keys, k)
		}

		it.Close()
		for _, k := range keys {
			tx.Delete(k)
		}

		return nil
	}) == badger.ErrConflict {
	}

	log.WithFields(log.Fields{
		"acc": acc,
	}).Debug("BDGR DeleteAccount")
}

func (s *BadgerStorage) GetOffset(acc AccountID, thr AccountID) int {
	k := kActiveThread(acc, thr)
	var r int
	for s.db.View(func(tx *badger.Txn) error {
		item, err := tx.Get(k)
		if err == badger.ErrKeyNotFound {
			r = -1
			return nil
		}

		v, _ := item.Value()
		r, _ = strconv.Atoi(string(v))

		return nil
	}) == badger.ErrConflict {
	}

	log.WithFields(log.Fields{
		"acc": acc,
		"thr": thr,
		"r":   r,
	}).Debug("BDGR GetOffset")

	return r
}

func (s *BadgerStorage) UpdateOffset(acc AccountID, thr ThreadID,
	offset int) bool {
	r := false
	k := kActiveThread(acc, thr)
	for s.db.Update(func(tx *badger.Txn) error {
		_, err := tx.Get(k)
		if err != nil {
			return nil
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

	log.WithFields(log.Fields{
		"acc":    acc,
		"thr":    thr,
		"offset": offset,
		"r":      r,
	}).Debug("BDGR UpdateOffset")

	return r
}

func kWebm(url string) []byte {
	return []byte(pW + sP + url)
}

func (s *BadgerStorage) GetWebm(url string) string {
	var r string
	for s.db.View(func(tx *badger.Txn) error {
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
	}) == badger.ErrConflict {
	}

	log.WithFields(log.Fields{
		"url": url,
		"r":   r,
	}).Debug("BDGR GetWebm")

	return r
}

func (s *BadgerStorage) UpdateWebm(url string, prev string, curr string) bool {
	r := false
	k := kWebm(url)
	for s.db.Update(func(tx *badger.Txn) error {
		var v []byte
		item, err := tx.Get(k)
		switch err {
		case badger.ErrKeyNotFound:
			v = []byte(webm.NotFound)

		case nil:
			v, err = item.Value()
			if err != nil {
				return nil
			}

		default:
			return nil
		}

		if prev == string(v) {
			tx.SetWithTTL(k, []byte(curr), s.config.WebmTTL)
			r = true
		}

		return nil
	}) == badger.ErrConflict {
	}

	log.WithFields(log.Fields{
		"url":  url,
		"prev": prev,
		"curr": curr,
		"r":    r,
	}).Debug("BDGR UpdateWebm")

	return r
}

func (s *BadgerStorage) DeleteOrphanWebms() {
	s.db.Update(func(tx *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := tx.NewIterator(opts)
		defer it.Close()
		prefix := []byte(pW)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			k := item.Key()
			v, err := item.Value()
			if err != nil {
				return err
			}

			if string(v) == webm.Pending {
				tx.Delete(k)
			}
		}

		return nil
	})
}

func (s *BadgerStorage) Close() error {
	return s.db.Close()
}
