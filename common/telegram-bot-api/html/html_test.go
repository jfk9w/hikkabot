package html

import (
	"testing"

	"github.com/jfk9w-go/hikkabot/common/gox/testx"
)

const HtmlSample = "<strong>Революция во Франции тред #5</strong>" +
	"<br><br>В Париже на Елисейских полях третью субботу подряд продолжаются протесты &quot;желтых жилетов&quot; " +
	"против повышения цен на топливо и налоговой политики правительства. Сотни манифестантов оккупировали " +
	"исторический центр французской столицы. Они жгут автомобили, возводят баррикады, забрасывают полицейских " +
	"камнями и самодельными снарядами с желтой краской." +
	"<br><br>Задержаны более 270 человек, около 120 человек пострадали" +
	"<br><br>Старые треды: " +
	"<br><span class=\"spoiler\">1 <a href=\"https:&#47;&#47;2ch.hk&#47;b&#47;res&#47;187447026.html\" target=\"_blank\" rel=\"nofollow noopener noreferrer\">https:&#47;&#47;2ch.hk&#47;b&#47;res&#47;187447026.html</a>" +
	"<br>2 <a href=\"https:&#47;&#47;2ch.hk&#47;b&#47;res&#47;187456304.html\" target=\"_blank\" rel=\"nofollow noopener noreferrer\">https:&#47;&#47;2ch.hk&#47;b&#47;res&#47;187456304.html</a>" +
	"<br>3 <a href=\"https:&#47;&#47;2ch.hk&#47;b&#47;res&#47;187466789.html\" target=\"_blank\" rel=\"nofollow noopener noreferrer\">https:&#47;&#47;2ch.hk&#47;b&#47;res&#47;187466789.html</a>" +
	"<br>4 <a href=\"https:&#47;&#47;2ch.hk&#47;b&#47;res&#47;187474805.html\" target=\"_blank\" rel=\"nofollow noopener noreferrer\">https:&#47;&#47;2ch.hk&#47;b&#47;res&#47;187474805.html</a></span>"

func TestFormat_Format(t *testing.T) {
	var assert = testx.Assert(t)
	assert.Equals([]string{
		"<strong>Революция во</strong>",
	}, NewFormat(1, 23, nil, nil).SetDefaults().Format(HtmlSample))
}
