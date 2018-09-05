package feed

import (
	"strings"

	"sort"

	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/red"
	. "github.com/pkg/errors"
)

type Service interface {
	Load(*State) (Load, error)
	Title(*State) string
}

type GenericService struct {
	Typed map[Type]Service
}

func (service *GenericService) Load(state *State) (Load, error) {
	var concrete, ok = service.Typed[state.Type]
	if ok {
		var load, err = concrete.Load(state)
		if err != nil {
			return &DummyLoad{}, err
		}

		return load, nil
	}

	return &DummyLoad{}, Errorf("unsupported type: %s", state.Type)
}

func (service *GenericService) Title(state *State) string {
	var concrete, ok = service.Typed[state.Type]
	if ok {
		return concrete.Title(state)
	}

	return string(state.Type)
}

type DvachService struct {
	Dvach
	Aconvert
}

func ParseDvachRef(value string) (dvach.Ref, error) {
	var tokens = strings.Split(value, "/")
	if len(tokens) != 2 {
		return dvach.Ref{}, Errorf("invalid thread ID: %s", value)
	}

	return dvach.ToRef(tokens[0], tokens[1])
}

func (service *DvachService) ParseState(state *State) (ref dvach.Ref, meta *DvachMeta, err error) {
	ref, err = ParseDvachRef(state.ID)
	if err != nil {
		return
	}

	meta = new(DvachMeta)
	err = state.ParseMeta(meta)
	return
}

func (service *DvachService) Load(state *State) (Load, error) {
	var (
		ref    dvach.Ref
		offset = state.Offset
		meta   *DvachMeta
		posts  []*dvach.Post
		err    error
	)

	ref, meta, err = service.ParseState(state)
	if err != nil {
		return nil, err
	}

	if offset > 0 {
		offset += 1
	}

	posts, err = service.Dvach.Posts(ref, offset)
	if err != nil {
		return nil, err
	}

	return &DvachLoad{service.Dvach, service.Aconvert, meta, posts, 0}, nil
}

func (service *DvachService) Title(state *State) string {
	var (
		meta = new(DvachMeta)
		err  error
	)

	err = state.ParseMeta(meta)
	if err != nil {
		return string(state.Type)
	}

	return meta.Title
}

type RedService struct {
	Red
}

func (service *RedService) Load(state *State) (Load, error) {
	var (
		offset = float32(state.Offset)
		meta   = new(RedMeta)
		data   []red.ThingData
		err    error
	)

	err = state.ParseMeta(meta)
	if err != nil {
		return nil, err
	}

	data, err = service.Listing(state.ID+"/"+meta.Mode, 100)
	if err != nil {
		return nil, err
	}

	sort.Sort(SortRedData(data))
	for i, datum := range data {
		if datum.CreatedUTC > offset {
			return &RedLoad{service.Red, data, i}, nil
		}
	}

	return &DummyLoad{}, nil
}

func (service *RedService) Title(state *State) string {
	return state.ID
}

type SortRedData []red.ThingData

func (data SortRedData) Len() int {
	return len(data)
}

func (data SortRedData) Less(i, j int) bool {
	return data[i].CreatedUTC < data[j].CreatedUTC
}

func (data SortRedData) Swap(i, j int) {
	data[i], data[j] = data[j], data[i]
}
