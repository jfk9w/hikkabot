package text

import (
	"strconv"
	"testing"

	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/misc"
	"github.com/jfk9w-go/testx"
)

func TestFormatThread_176683748(t *testing.T) {
	assert := testx.Assert(t)
	thread := new(dvach.Thread)
	if err := misc.ReadJSON("testdata/176683748.json", thread); err != nil {
		t.Fatal(err)
	}

	thread.Num, _ = strconv.Atoi(thread.NumString)
	thread.Board = "b"

	threads := searchThreads([]*dvach.Thread{thread}, []string{})

	assert.Equals(`<b>28/05/18 Пнд 10:08:00</b>
#B176683748
0 / 0.00/hr
---
Ну это пиздец, господа. Скоро мне кажется правительство сформирует бригады, которые  будут просто ходить по квартирам и забирать какой-то процент от найденных в доме денег. 

<i>&gt;Правительство обсуждает повышение НДС с 18% до 20%, что может принести бюджету около двух </i>`,
		FormatThread(threads[0]),
	)
}
