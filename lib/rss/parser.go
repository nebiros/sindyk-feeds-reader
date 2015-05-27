package rss

import (
	"io/ioutil"
	"encoding/xml"
	"html/template"
	"net/http"
	"errors"
	"bytes"
	"github.com/nebiros/sindyk-feeds-reader/lib/charset"
)

type Rss struct {
	XMLName xml.Name `xml:"rss"`
	Version string `xml:"version,attr"`
	// Required
	Title string `xml:"channel>title"`
	Link string `xml:"channel>link"`
	Description string `xml:"channel>description"`
	// Optional
	PubDate string `xml:"channel>pubDate"`
	ItemList []Item `xml:"channel>item"`
}

type Item struct {
	// Required
	Title string `xml:"title"`
	Link string `xml:"link"`
	Description template.HTML `xml:"description"`
	// Optional
	Content template.HTML `xml:"encoded"`
	PubDate string `xml:"pubDate"`
	Comments string `xml:"comments"`
	Guid string `xml:"guid"`
	DcSubject string `xml:"http://purl.org/dc/elements/1.1/ subject"`
	DcCreator template.HTML `xml:"http://purl.org/dc/elements/1.1/ creator"`
	Enclosure Enclosure `xml:"enclosure"`
	Category string `xml:"category"`
	Hour string `xml:"hora"`
	Order string `xml:"order"`
	Id string `xml:"id"`
}

type Enclosure struct {
	Url string `xml:"url,attr"`
	MimeType string `xml:"type,attr"`
}

type RssHandlerFunc func (rss Rss, err error)

func Fetch(uri string, rh RssHandlerFunc) {
	_ = "breakpoint"
	b, err := LoadUri(uri)
	if err != nil {
		if rh != nil {
			rh(Rss{}, err)
		}
		return
	}

	f, err := Parse(b)
	if err != nil {
		if rh != nil {
			rh(Rss{}, err)
		}
		return
	}

	if rh != nil {
		rh(f, nil)
	}
}

func LoadUri(uri string) (content []byte, err error) {
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

func Parse(c []byte) (feed Rss, err error) {
	r := Rss{}

	decoder := xml.NewDecoder(bytes.NewReader(c))
	decoder.CharsetReader = charset.CharsetReader
	err = decoder.Decode(&r)
	if err != nil {
		return Rss{}, err
	}

	if r.Version != "2.0" {
		return Rss{}, errors.New("Not a valid RSS 2.0 feed")
	}

	for i, _ := range r.ItemList {
		if r.ItemList[i].Content != "" {
			r.ItemList[i].Description = r.ItemList[i].Content
		}
	}
	return r, nil
}