package reader

type Params struct {
	Address, Username, Password, Database, Port, Charset string
}

type Feed struct {
	Id int
	Url string
	Active bool
}

func Start(p Params) {
	OpenDb(p)

	// get all feeds from db.
	feeds := Feeds()
	// process each feed.
	PullFeeds(feeds)
}
