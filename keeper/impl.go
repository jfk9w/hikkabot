package keeper

import (
	"encoding/json"

	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/telegram"
	"github.com/orcaman/concurrent-map"
)

type keeper struct {
	offsets cmap.ConcurrentMap
}

func NewKeeper() Json {
	return &keeper{
		offsets: cmap.New(),
	}
}

func (k *keeper) MarshalJSON() ([]byte, error) {
	state := make(map[string]interface{})
	state["offsets"] = k.offsets
	return json.MarshalIndent(state, "", "  ")
}

func (k *keeper) UnmarshalJSON(data []byte) error {
	state := make(map[string]json.RawMessage)
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}

	if bytes, ok := state["offsets"]; ok {
		offsets := make(map[string]int)
		if err := json.Unmarshal(bytes, &offsets); err != nil {
			return err
		}

		for key, value := range offsets {
			k.offsets.Set(key, value)
		}
	}

	return nil
}

func (k *keeper) SetOffset(chat telegram.ChatRef, thread dvach.Ref, offset int) {
	k.offsets.Set(refs2key(chat, thread), offset)
}

func (k *keeper) DeleteOffset(chat telegram.ChatRef, thread dvach.Ref) {
	k.offsets.Remove(refs2key(chat, thread))
}

func (k *keeper) GetOffsets() Offsets {
	offsets := make(Offsets)
	for item := range k.offsets.IterBuffered() {
		chat, thread := key2refs(item.Key)
		offset := item.Val.(int)

		chatOffsets, ok := offsets[chat]
		if !ok {
			chatOffsets = make(map[dvach.Ref]int)
			offsets[chat] = chatOffsets
		}

		chatOffsets[thread] = offset
	}

	return offsets
}
