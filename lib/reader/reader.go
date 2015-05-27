package reader

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"runtime"
	"fmt"
	"github.com/nebiros/sindyk-feeds-reader/lib/rss"
)

const (
	// "user:password@tcp(localhost:3306)/dbname??charset=utf8"
	DbDefaultDsnFormat = "%s:%s@tcp(%s:%s)/%s?charset=%s"
	DbDefaultDriver = "mysql"
)

var (
	// db connection.
	DB *sql.DB
	// feeds channel.
	feedsChannel chan rss.Rss
)

type Params struct {
	Address, Username, Password, Database, Port, Charset string
}

type FeedRow struct {
	Id int
	Url string
	Active bool
}

func Start(p Params) {
	OpenDb(p)

	// get all feeds from db.
	feeds := Feeds()
	// process each feed.
	Process(feeds)
}

func OpenDb(p Params) {
	var err error

	dsn := buildDbDsn(p)
	DB, err = sql.Open(DbDefaultDriver, dsn)
	if err != nil {
		log.Fatal(err)
	}

	err = DB.Ping()
	if err != nil {
		log.Fatal(err)
	}
}

func buildDbDsn(p Params) string {
	dsn := fmt.Sprintf(DbDefaultDsnFormat,
		p.Username,
		p.Password,
		p.Address,
		p.Port,
		p.Database,
		p.Charset);

	return dsn
}

func Feeds() (feeds []FeedRow) {
	query := `SELECT
		feeds.id AS feed_id,
		feeds.active AS feed_active,
		feeds.link AS url
		FROM feeds
		LEFT JOIN
		sections
		ON
		sections.id = feeds.section_id
		AND sections.active = ?
		AND sections.external = ?
		AND sections.other <> ?
		WHERE
		feeds.active = ?`

	stmt, err := DB.Prepare(query)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(1, 0, 1, 1)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		f := FeedRow{}

		err := rows.Scan(&f.Id, &f.Active, &f.Url)
		if err != nil {
			log.Fatal(err)
		}

		feeds = append(feeds, f)
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	defer DB.Close()

	return feeds
}

func Process(feeds []FeedRow) {
	_ = "breakpoint"

	runtime.GOMAXPROCS(runtime.NumCPU())

	for _, f := range feeds {
		go rss.Fetch(f.Url, rssHandler)
	}

	feedsChannel = make(chan rss.Rss)
	for r := range feedsChannel {
		fmt.Printf("[PullFeeds] r, %T, %#v\n", r, r)
	}
}

func rssHandler(rss rss.Rss, err error) {
	feedsChannel <- rss
}
