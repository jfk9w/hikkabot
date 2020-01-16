package main

import (
	"io/ioutil"
	"log"
	"os"
	"regexp"

	"github.com/otiai10/gosseract/v2"
)

type OCR interface {
}

func main() {
	client := gosseract.NewClient()
	defer client.Close()
	client.SetLanguage("rus", "eng")
	file, err := os.Open("/Users/ikulkov/Downloads/Telegram Desktop/photo_2020-01-15_16-22-26.jpg")
	if err != nil {
		panic(err)
	}
	data, err := ioutil.ReadAll(file)
	file.Close()
	if err != nil {
		panic(err)
	}
	err = client.SetImageFromBytes(data)
	if err != nil {
		panic(err)
	}
	text, err := client.Text()
	if err != nil {
		panic(err)
	}
	log.Println(text)
	log.Println(regexp.MustCompile(`(?is).*?(твоя.*?ма(ть|ма).*?(умр(е|ё)т|сдохнет)|mother.*?will.*?die|проклят|curse).*`).MatchString(text))
}
