package format

import (
	"reflect"
	"runtime/debug"
	"testing"

	telegram "github.com/jfk9w-go/telegram-bot-api"
)

func assertEquals(t *testing.T, actual, expected interface{}) {
	if !reflect.DeepEqual(expected, actual) {
		debug.PrintStack()
		t.Errorf("Values are not equal.\nExpected: %+v\nActual:   %+v", expected, actual)
	}
}

func Test_PageWriter_SingleLetter(t *testing.T) {
	pw := NewHTML(1, 0, nil, nil)
	pw.writeBreakable("hello")
	pw.writeBreakable("hello")
	pw.writeUnbreakable("hello")
	assertEquals(t,
		[]string{"h", "e", "l", "l", "o", "h", "e", "l", "l", "o", "Bytes", "R", "O", "K", "E", "N"},
		pw.Format().Pages)
}

func Test_PageWriter_OnePage(t *testing.T) {
	pw := NewHTML(10, 1, nil, nil)
	pw.writeBreakable("Hello, Mark. How do you do?")
	assertEquals(t, []string{"Hello,"}, pw.Format().Pages)
}

func Test_PageWriter_ManyPages(t *testing.T) {
	pw := NewHTML(10, -1, nil, nil)
	pw.writeBreakable("Hello, Mark. How do you do?")
	assertEquals(t, []string{"Hello,", "Mark. How", "do you do?"}, pw.Format().Pages)
}

func Test_PageWriter_Emojis(t *testing.T) {
	pw := NewHTML(3, 0, nil, nil)
	pw.writeBreakable("😭👌🎉😞😘😔")
	assertEquals(t, []string{"😭👌🎉", "😞😘😔"}, pw.Format().Pages)
}

func Test_PageWriter_Unbreakable(t *testing.T) {
	pw := NewHTML(8, 0, nil, nil)
	pw.writeUnbreakable("123")
	pw.writeUnbreakable("😭👌🎉😞😘😔")
	assertEquals(t, []string{"123", "😭👌🎉😞😘😔"}, pw.Format().Pages)
}

type testLinkPrinter struct{}

func (testLinkPrinter) Print(link *Link) string {
	if href, ok := link.Attr("href"); ok {
		return href
	} else {
		return ""
	}
}

func Test_PageWriter_BasicHTML(t *testing.T) {
	pages := NewHTML(72, 0, nil, nil).
		Tag("b").
		Text("A Study in Scarlet is an 1887 detective novel by Scottish author Arthur Conan Doyle. ").
		EndTag().
		Link("Wikipedia", "https://en.wikipedia.org/wiki/A_Study_in_Scarlet").
		Format().Pages
	sample := []string{
		`<b>A Study in Scarlet is an 1887 detective novel by Scottish author</b>`,
		`<b>Arthur Conan Doyle. </b>`,
		`<a href="https://en.wikipedia.org/wiki/A_Study_in_Scarlet">Wikipedia</a>`,
	}
	assertEquals(t, pages, sample)

	pages = NewHTML(0, 0, nil, nil).
		Parse(`<strong>Музыкальный webm mp4 тред</strong><br><em>Не нашел - создал</em><br>Делимся вкусами, ищем музыку, создаем, нарезаем, постим свои либо чужие музыкальные видео.<br>Рекомендации для самостоятельного поиска соусов: <a href="https:&#47;&#47;pastebin.com&#47;i32h11vd" target="_blank" rel="nofollow noopener noreferrer">https:&#47;&#47;pastebin.com&#47;i32h11vd</a>`).
		Format().Pages
	sample = []string{
		`<b>Музыкальный webm mp4 тред</b>
<i>Не нашел - создал</i>
Делимся вкусами, ищем музыку, создаем, нарезаем, постим свои либо чужие музыкальные видео.
Рекомендации для самостоятельного поиска соусов: <a href="https://pastebin.com/i32h11vd">https://pastebin.com/i32h11vd</a>`,
	}
	assertEquals(t, pages, sample)

	pages = NewHTML(75, 0, nil, nil).
		Parse(`<strong>Музыкальный webm mp4 тред</strong><br><em>Не нашел - создал</em><br>Делимся вкусами, ищем музыку, создаем, нарезаем, постим свои либо чужие музыкальные видео.<br>Рекомендации для самостоятельного поиска соусов: <a href="https:&#47;&#47;pastebin.com&#47;i32h11vd" target="_blank" rel="nofollow noopener noreferrer">https:&#47;&#47;pastebin.com&#47;i32h11vd</a>`).
		Format().Pages
	sample = []string{
		`<b>Музыкальный webm mp4 тред</b>
<i>Не нашел - создал</i>
Делимся вкусами,`,
		`ищем музыку, создаем, нарезаем, постим свои либо чужие музыкальные видео.`,
		`Рекомендации для самостоятельного поиска соусов: `,
		`<a href="https://pastebin.com/i32h11vd">https://pastebin.com/i32h11vd</a>`,
	}
	assertEquals(t, pages, sample)
}

func Test_PageWriter_LinkPrinter(t *testing.T) {
	pages := NewHTML(50, 0, nil, testLinkPrinter{}).
		Parse(`<strong>Музыкальный webm mp4 тред</strong><br><em>Не нашел - создал</em><br>Делимся вкусами, ищем музыку, создаем, нарезаем, постим свои либо чужие музыкальные видео.<br>Рекомендации для самостоятельного поиска соусов: <a href="https:&#47;&#47;pastebin.com&#47;i32h11vd" target="_blank" rel="nofollow noopener noreferrer">https:&#47;&#47;pastebin.com&#47;i32h11vd</a>`).
		Format().Pages
	sample := []string{
		`<b>Музыкальный webm mp4 тред</b>
<i>Не нашел -</i>`,
		`<i>создал</i>
Делимся вкусами, ищем музыку,`,
		`создаем, нарезаем, постим свои либо чужие`,
		`музыкальные видео.
Рекомендации для`,
		`самостоятельного поиска соусов: `,
		`https://pastebin.com/i32h11vd`,
	}
	assertEquals(t, pages, sample)
}

func Test_PageWriter_LinkInTag(t *testing.T) {
	pages := NewHTML(telegram.MaxMessageSize, 0, nil, nil).
		Parse(
			`<i>&gt;Почему товарищ Майор ничего не делает с рабочими домами, ведь следят они ужасно, в том же вконтактике <a href="https://vk.com/pedestrian111,">https://vk.com/pedestrian111,</a> да и симпатию питать к таким сложно? Заносят? Никто не жалуется, а план легче на наркошах выполнять?
    Делает, но о подвигах все молчат с 2011 года? Я чего-то не понимаю и это все норма? </i>`).
		Format().Pages
	sample := []string{
		`<i>&gt;Почему товарищ Майор ничего не делает с рабочими домами, ведь следят они ужасно, в том же вконтактике </i><a href="https://vk.com/pedestrian111,">https://vk.com/pedestrian111,</a><i> да и симпатию питать к таким сложно? Заносят? Никто не жалуется, а план легче на наркошах выполнять?
    Делает, но о подвигах все молчат с 2011 года? Я чего-то не понимаю и это все норма? </i>`,
	}
	assertEquals(t, pages, sample)
}
