package reader

import (
	"io/ioutil"
	"encoding/xml"
	"html/template"
	"log"
	"net/http"
	"errors"
	"fmt"
	"bytes"
	"github.com/nebiros/sindyk-feeds-reader/lib/charset"
)

type Rss2 struct {
	XMLName xml.Name `xml:"rss"`
	Version string `xml:"version,attr"`
	// Required
	Title string `xml:"channel>title"`
	Link string `xml:"channel>link"`
	Description string `xml:"channel>description"`
	// Optional
	PubDate string `xml:"channel>pubDate"`
	ItemList []Rss2Item `xml:"channel>item"`
}

type Rss2Item struct {
	// Required
	Title string `xml:"title"`
	Link string `xml:"link"`
	Description template.HTML `xml:"description"`
	// Optional
	Content template.HTML `xml:"encoded"`
	PubDate string `xml:"pubDate"`
	Comments string `xml:"comments"`
	Guid string `xml:"guid"`
	Category string `xml:"category"`
	Hour string `xml:"hora"`
	Order string `xml:"order"`
	Id string `xml:"id"`
}

func FetchRss2(uri string) (feed Rss2, err error) {
	_ = "breakpoint"
	b, err := LoadRss2Uri(uri)
	if err != nil {
		log.Println(err)
		return Rss2{}, err
	}

	f, err := ParseRss2Content(b)
	if err != nil {
		log.Println(err)
		return Rss2{}, err
	}
	fmt.Printf("f, %T, %#v\n", f, f)
	return f, nil
}

func LoadRss2Uri(uri string) (content []byte, err error) {
	client := http.DefaultClient
	resp, err := client.Get(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return []byte(data), nil
}

func ParseRss2Content(c []byte) (feed Rss2, err error) {
	r := Rss2{}

	decoder := xml.NewDecoder(bytes.NewReader(c))
	decoder.CharsetReader = charset.CharsetReader
	err = decoder.Decode(&r)
	if err != nil {
		return Rss2{}, err
	}

	if r.Version == "2.0" {
		// RSS 2.0
		for i, _ := range r.ItemList {
			if r.ItemList[i].Content != "" {
				r.ItemList[i].Description = r.ItemList[i].Content
			}
		}
		return r, nil
	}

	return Rss2{}, errors.New("Not a valid RSS 2.0 feed")
}