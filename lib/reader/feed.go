package reader

import (
	// "fmt"
	"runtime"
	// "sync"
	// "log"
	// rss "github.com/jteeuwen/go-pkg-rss"
	// "github.com/nebiros/sindyk-feeds-reader/lib/charset"
)

// var pendingItems chan *rss.Item

func PullFeeds(fds []Feed) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	// var wg sync.WaitGroup

	// feed := rss.New(5, true, chanHandler, itemHandler)
	//
	// for _, f := range fds {
	// 	// wg.Add(1)
	//
	// 	// go PullFeed(feed, f.Url, &wg)
	// 	go PullFeed(feed, f.Url)
	// }
	//
	// // wg.Wait()
	//
	// pendingItems = make(chan *rss.Item)
	//
	// for i := range pendingItems {
	// 	// wg.Add(1)
	// 	fmt.Printf("[PullFeeds] i, %T, %#v\n", i, i)
	// 	// wg.Done()
	// }

	// wg.Wait()
	_ = "breakpoint"
	for _, f := range fds {
		// go FetchRss2(f.Url)
		FetchRss2(f.Url)
	}
}

/*
// func PullFeed(feed *rss.Feed, uri string, wg *sync.WaitGroup) {
func PullFeed(feed *rss.Feed, uri string) {
	// fmt.Printf("[fetchFeed] feed %s, %T, %#v\n", feed.Url, feed, feed)
	if err := feed.Fetch(uri, charset.CharsetReader); err != nil {
		log.Printf("[fetchFeed error] %s, %s", feed.Url, err)
		// wg.Done(); return;
	}

	// wg.Done()
}

func chanHandler(feed *rss.Feed, newchannels []*rss.Channel) {
	// fmt.Printf("%d new channel(s) in %s\n", len(newchannels), feed.Url)
	// fmt.Printf("[chanHandler] feed %s, %T, %#v\n", feed.Url, feed, feed)
	// fmt.Printf("[chanHandler] newchannels, %T, %#v\n", newchannels, newchannels)
}

func itemHandler(feed *rss.Feed, ch *rss.Channel, newitems []*rss.Item) {
	// fmt.Printf("%d new item(s) in %s\n", len(newitems), feed.Url)
	// fmt.Printf("[itemHandler] feed %s, %T, %#v\n", feed.Url, feed, feed)
	// fmt.Printf("[itemHandler] ch, %T, %#v\n", ch, ch)
	// fmt.Printf("[itemHandler] newitems, %T, %#v\n", newitems, newitems)
	for _, i := range newitems {
		pendingItems <- i
	}
}
*/
