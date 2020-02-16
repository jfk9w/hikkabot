package media

import "regexp"

type OCR struct {
	Languages []string
	Regex     *regexp.Regexp
}

type OCRClient interface {
	SetImage(path string) error
	SetImageFromBytes(data []byte) error
}
