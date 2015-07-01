package reader

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/nebiros/sindyk-feeds-reader/lib/charset"
)

type Rss struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	// Required
	Title       string `xml:"channel>title"`
	Link        string `xml:"channel>link"`
	Description string `xml:"channel>description"`
	// Optional
	PubDate     string     `xml:"channel>pubDate"`
	RssItemList []*RssItem `xml:"channel>item"`
}

type RssItem struct {
	// Required
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	// Optional
	Content          string            `xml:"encoded"`
	PubDate          string            `xml:"pubDate"`
	Comments         string            `xml:"comments"`
	Guid             string            `xml:"guid"`
	DcSubject        string            `xml:"http://purl.org/dc/elements/1.1/ dc:subject"`
	DcCreator        string            `xml:"http://purl.org/dc/elements/1.1/ dc:creator"`
	RssItemEnclosure *RssItemEnclosure `xml:"enclosure"`
	Category         string            `xml:"category"`
	Hour             string            `xml:"hora"`
	Order            int               `xml:"order"`
	Id               int               `xml:"id"`
	Related          string            `xml:"relacionadas"`
	Subject          string            `xml:"subject"`
	Creator          string            `xml:"creator"`
}

type RssItemEnclosure struct {
	Url      string `xml:"url,attr"`
	MimeType string `xml:"type,attr"`
}

func FetchRss(uri string) (rss *Rss, err error) {
	b, err := LoadRssUri(uri)
	if err != nil {
		return nil, err
	}

	f, err := ParseRss(b)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func LoadRssUri(uri string) (content []byte, err error) {
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

func ParseRss(c []byte) (rss *Rss, err error) {
	r := new(Rss)

	decoder := xml.NewDecoder(bytes.NewReader(c))
	decoder.CharsetReader = charset.CharsetReader
	err = decoder.Decode(r)
	if err != nil {
		return nil, err
	}

	if r.Version != "2.0" {
		return nil, errors.New("Not a valid RSS 2.0 feed")
	}

	for i, _ := range r.RssItemList {
		if r.RssItemList[i].Content != "" {
			r.RssItemList[i].Description = r.RssItemList[i].Content
		}
	}
	return r, nil
}
