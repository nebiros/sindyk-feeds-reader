package reader

import (
	"fmt"
	"runtime"
	// "sync"
	// "log"
	// rss "github.com/jteeuwen/go-pkg-rss"
	// "github.com/nebiros/sindyk-feeds-reader/lib/charset"
)

var feeds chan Rss

func PullFeeds(fds []Feed) {
	_ = "breakpoint"

	runtime.GOMAXPROCS(runtime.NumCPU())

	for _, f := range fds {
		go FetchRss(f.Url, rssHandler)
	}

	feeds = make(chan Rss)
	for r := range feeds {
		fmt.Printf("[PullFeeds] r, %T, %#v\n", r, r)
	}
}

func rssHandler(rss Rss, err error) {
	feeds <- rss
}
