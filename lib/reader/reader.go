package reader

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"runtime"
	"fmt"
	"strings"
	"html"
	"net/url"
)

const (
	// "user:password@tcp(localhost:3306)/dbname??charset=utf8"
	DbDefaultDsnFormat = "%s:%s@tcp(%s:%s)/%s?charset=%s"
	DbDefaultDriver = "mysql"
)

var (
	// db connection.
	conn *sql.DB
	// feeds channel.
	feedsChannel chan feedChannelPack
)

type Params struct {
	Address, Username, Password, Database, Port, Charset string
}

type FeedRow struct {
	Id int
	Url string
	Active bool
}

type ItemRow struct {
	FeedId int
	ExternalId int
	Title string
	Description string
	Link string
	PubDate string
	Content string
	Creator string
	ImageUrl string
	Active bool
	DisplayOrder int
	Subject string
	Category string
	Hour string
	Related string
	Slug string
}

type feedChannelPack struct {
	rss *Rss
	feedRow *FeedRow
	err error
}

func Start(p Params) {
	OpenDb(p)

	// get all feeds from conn.
	activeFeeds := ActiveFeedsFromDb()
	// process each feed.
	Process(activeFeeds)

	defer conn.Close()
}

func OpenDb(p Params) {
	var err error

	conn, err = sql.Open(DbDefaultDriver, buildDbDsn(p))
	if err != nil {
		log.Fatal("[OpenDb] " + err.Error())
		conn.Close()
	}

	err = conn.Ping()
	if err != nil {
		log.Fatal("[OpenDb] " + err.Error())
		conn.Close()
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

func ActiveFeedsFromDb() (activeFeeds []*FeedRow) {
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

	rows, err := conn.Query(query, 1, 0, 1, 1)
	if err != nil {
		log.Fatal("[ActiveFeedsFromDb] " + err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		f := new(FeedRow)

		err := rows.Scan(&f.Id, &f.Active, &f.Url)
		if err != nil {
			log.Fatal("[ActiveFeedsFromDb] " + err.Error())
		}

		activeFeeds = append(activeFeeds, f)
	}
	err = rows.Err()
	if err != nil {
		log.Fatal("[ActiveFeedsFromDb] " + err.Error())
	}

	return activeFeeds
}

func Process(af []*FeedRow) {
	_ = "breakpoint"

	runtime.GOMAXPROCS(runtime.NumCPU())

	for _, f := range af {
		go FetchRss(f.Url, f, rssHandler)
	}

	feedsChannel = make(chan feedChannelPack)
	for cp := range feedsChannel {
		// fmt.Printf("[PullFeeds] r, %T, %#v\n\n", cp, cp)
		if cp.err != nil {
			log.Println("[Process] " + cp.err.Error())
		} else {
			for _, i := range cp.rss.RssItemList {
				// item := cp.rss.RssItemList[l]
				fmt.Printf("[RssItemList] item, %T, %#v\n\n", i, i)
				content := i.Content
				if len(content) <= 0 {
					content = i.Description
				}

				subject := strings.TrimSpace(i.Subject)
				if len(subject) <= 0 {
					subject = strings.TrimSpace(i.DcSubject)
				}

				creator := i.Creator
				if len(creator) <= 0 {
					creator = i.DcCreator
				}

				var imageUrl string
				if i.RssItemEnclosure != nil {
					mime := strings.Split(i.RssItemEnclosure.MimeType, "/")
					if mime[0] == "image" {
						imageUrl = i.RssItemEnclosure.Url
					}
				}

				link := strings.TrimSpace(i.Link)

				u, err := url.Parse(link)
			    if err != nil {
			        log.Println("[Process] " + err.Error())
			    }

				ir := ItemRow{FeedId: cp.feedRow.Id,
					ExternalId: i.Id,
					Title: strings.TrimSpace(i.Title),
					Description: html.EscapeString(i.Description),
					Link: link,
					PubDate: strings.TrimSpace(i.PubDate),
					Content: html.EscapeString(content),
					Creator: html.EscapeString(creator),
					ImageUrl: imageUrl,
					Active: true,
					DisplayOrder: i.Order,
					Category: strings.TrimSpace(i.Category),
					Subject: subject,
					Hour: strings.TrimSpace(i.Hour),
					Related: strings.TrimSpace(i.Related),
					Slug: u.Path}
				// SaveItemToDb(&ir)
			}
		}
	}
}

func rssHandler(r *Rss, fr *FeedRow, err error) {
	// DisableFeedItemsFromDb(fr.Id)
	fmt.Printf("[rssHandler] fr, %T, %#v\n\n", fr, fr)
	feedsChannel <- feedChannelPack{rss: r, feedRow: fr, err: err}
}

func SaveItemToDb(ir *ItemRow) {
	var (
		itemIdSelectQuery string
		itemId *int
	)
	_ = "breakpoint"
	if ir.ExternalId > 0 {
		itemIdSelectQuery = `SELECT
            items.id
            FROM
            items
            WHERE
            items.external_id = ?
            AND
            items.feed_id = ?`

		err := conn.QueryRow(itemIdSelectQuery, ir.ExternalId, ir.FeedId).Scan(&itemId)
		if err != nil {
			if err != sql.ErrNoRows {
				log.Fatal("[SaveItemToDb] " + err.Error())
			}
		}
	} else {
		itemIdSelectQuery = `SELECT
		    items.id
		    FROM
		    items
		    WHERE
		    items.title LIKE ?
		    AND
		    items.feed_id = ?`

		err := conn.QueryRow(itemIdSelectQuery, fmt.Sprintf("%%s%", ir.Title), ir.FeedId).Scan(itemId)
		if err != nil {
			if err != sql.ErrNoRows {
				log.Fatal("[SaveItemToDb] " + err.Error())
			}
		}
	}
	fmt.Printf("[SaveItemToDb] itemId, %T, %#v\n\n", itemId, itemId)

	tx, err := conn.Begin()
	if err != nil {
		log.Fatal("[SaveItemToDb] " + err.Error())
	}
	defer tx.Rollback()

	if itemId != nil {
		updateQuery := `UPDATE items SET external_id = ?,
			feed_id = ?,
			title = ?,
			link = ?,
			pubdate = ?,
			creator = ?,
			display_order = ?,
			subject = ?,
			category = ?,
			description = ?,
			content = ?,
			image = ?,
			hour = ?,
			relacionadas = ?,
			active = ?
			WHERE
			id = ?`

		stmt, err := tx.Prepare(updateQuery)
		if err != nil {
			log.Fatal("[SaveItemToDb] " + err.Error())
		}
		defer stmt.Close()

		_, err = stmt.Exec(ir.ExternalId,
			ir.FeedId,
			ir.Title,
			ir.Link,
			ir.PubDate,
			ir.Creator,
			ir.DisplayOrder,
			ir.Subject,
			ir.Category,
			ir.Description,
			ir.Content,
			ir.ImageUrl,
			ir.Hour,
			ir.Related,
			ir.Active,
			itemId)
		if err != nil {
			log.Fatal("[SaveItemToDb] " + err.Error())
		}
	} else {
		insertQuery := `INSERT INTO items (external_id,
			feed_id,
			title,
			link,
			pubdate,
			creator,
			display_order,
			subject,
			category,
			description,
			content,
			image,
			hour,
			relacionadas,
			active)
			VALUES
			(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

		stmt, err := tx.Prepare(insertQuery)
		if err != nil {
			log.Fatal("[SaveItemToDb] " + err.Error())
		}
		defer stmt.Close()

		_, err = stmt.Exec(ir.ExternalId,
			ir.FeedId,
			ir.Title,
			ir.Link,
			ir.PubDate,
			ir.Creator,
			ir.DisplayOrder,
			ir.Subject,
			ir.Category,
			ir.Description,
			ir.Content,
			ir.ImageUrl,
			ir.Hour,
			ir.Related,
			ir.Active)
		if err != nil {
			log.Fatal("[SaveItemToDb] " + err.Error())
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Fatal("[SaveItemToDb] " + err.Error())
	}
}

func DisableFeedItemsFromDb(id int) {
	tx, err := conn.Begin()
	if err != nil {
		log.Fatal("[DisableFeedItemsFromDb] " + err.Error())
	}
	defer tx.Rollback()

	updateQuery := `UPDATE items SET active = 0
		WHERE
		feed_id = ?`

	stmt, err := tx.Prepare(updateQuery)
	if err != nil {
		log.Fatal("[DisableItemsFromDb] " + err.Error())
	}
	defer stmt.Close()

	_, err = stmt.Exec(id)
	if err != nil {
		log.Fatal("[DisableItemsFromDb] " + err.Error())
	}

	err = tx.Commit()
	if err != nil {
		log.Fatal("[DisableItemsFromDb] " + err.Error())
	}
}
