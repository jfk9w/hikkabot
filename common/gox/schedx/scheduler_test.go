package schedx

import (
	"fmt"
	"testing"
	"time"

	"encoding/json"

	"github.com/jfk9w-go/hikkabot/common/gox/syncx"
)

func TestScheduler(t *testing.T) {
	key := "test"
	state := syncx.NewMap()
	scheduler := New(1 * time.Second)
	scheduler.Init(func(id interface{}) {
		data, err := json.Marshal(state.Save(nil))
		if err != nil {
			panic(err)
		}

		fmt.Println(string(data))
	})

	scheduler.Schedule(key)

	time.Sleep(1 * time.Second)
	state.Put("1", 1)

	time.Sleep(1 * time.Second)
	state.Put("2", 2)

	time.Sleep(1 * time.Second)
	state.Put("3", 3)

	time.Sleep(1 * time.Second)
	if !scheduler.Cancel(key) {
		panic(nil)
	} else {
		fmt.Println("STOP")
	}

	time.Sleep(2 * time.Second)
}
