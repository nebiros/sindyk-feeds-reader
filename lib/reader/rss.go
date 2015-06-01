package reader

import (
	"io/ioutil"
	"encoding/xml"
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
	RssItemList []*RssItem `xml:"channel>item"`
}

type RssItem struct {
	// Required
	Title string `xml:"title"`
	Link string `xml:"link"`
	Description string `xml:"description"`
	// Optional
	Content string `xml:"encoded,omitempty"`
	PubDate string `xml:"pubDate,omitempty"`
	Comments string `xml:"comments,omitempty"`
	Guid string `xml:"guid,omitempty"`
	DcSubject string `xml:"http://purl.org/dc/elements/1.1/ dc:subject,omitempty"`
	DcCreator string `xml:"http://purl.org/dc/elements/1.1/ dc:creator,omitempty"`
	RssItemEnclosure *RssItemEnclosure `xml:"enclosure,omitempty"`
	Category string `xml:"category,omitempty"`
	Hour string `xml:"hora,omitempty"`
	Order int `xml:"order,omitempty"`
	Id int `xml:"id,omitempty"`
	Related string `xml:"relacionadas,omitempty"`
	Subject string `xml:"subject,omitempty"`
	Creator string `xml:"creator,omitempty"`
}

type RssItemEnclosure struct {
	Url string `xml:"url,attr"`
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