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
	Active int
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
	log.Println("[Start]")

	OpenDb(p)

	// get all feeds from conn.
	activeFeeds := ActiveFeedsFromDb()
	// process each feed.
	Process(activeFeeds)

	defer conn.Close()

	log.Println("[Done]")
}

func OpenDb(p Params) {
	var err error

	conn, err = sql.Open(DbDefaultDriver, buildDbDsn(p))
	if err != nil {
		log.Fatalf("[Error] [OpenDb] %s", err)
	}

	err = conn.Ping()
	if err != nil {
		log.Fatalf("[Error] [OpenDb] %s", err)
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
		log.Fatalf("[Error] [ActiveFeedsFromDb] %s", err)
	}
	defer rows.Close()

	for rows.Next() {
		f := new(FeedRow)

		err := rows.Scan(&f.Id, &f.Active, &f.Url)
		if err != nil {
			log.Fatalf("[Error] [ActiveFeedsFromDb] %s", err)
		}

		activeFeeds = append(activeFeeds, f)
	}
	err = rows.Err()
	if err != nil {
		log.Fatalf("[Error] [ActiveFeedsFromDb] %s", err)
	}

	return activeFeeds
}

func Process(af []*FeedRow) {
	_ = "breakpoint"

	runtime.GOMAXPROCS(runtime.NumCPU())

	for _, f := range af {
		log.Printf("[Process] %s\n", f.Url)

		go DisableFeedItemsFromDb(f.Id)
		go FetchRss(f.Url, f, rssHandler)
	}

	feedsChannel = make(chan feedChannelPack, len(af))
	defer close(feedsChannel)

	for cp := range feedsChannel {
		if cp.err != nil {
			log.Printf("[Error] [Process] %s\n", cp.err)
		} else {
			for _, i := range cp.rss.RssItemList {
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

				slug := u.Path
				if string([]rune(slug)[0]) == "/" {
					slug = slug[1:len(slug)]
				}

				log.Printf("[Item] [Start] %s\n", link)

				ir := ItemRow{FeedId: cp.feedRow.Id,
					ExternalId: i.Id,
					Title: strings.TrimSpace(i.Title),
					Description: html.EscapeString(i.Description),
					Link: link,
					PubDate: strings.TrimSpace(i.PubDate),
					Content: html.EscapeString(content),
					Creator: html.EscapeString(creator),
					ImageUrl: imageUrl,
					Active: 1,
					DisplayOrder: i.Order,
					Category: strings.TrimSpace(i.Category),
					Subject: subject,
					Hour: strings.TrimSpace(i.Hour),
					Related: strings.TrimSpace(i.Related),
					Slug: slug}
				SaveItemToDb(&ir)
			}
		}
	}
}

func rssHandler(r *Rss, fr *FeedRow, err error) {
	feedsChannel <-feedChannelPack{rss: r, feedRow: fr, err: err}
}

func SaveItemToDb(ir *ItemRow) (id int64) {
	var (
		itemIdSelectQuery string
		itemId int64
	)

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
				log.Fatalf("[Error] [SaveItemToDb] %s", err)
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

		err := conn.QueryRow(itemIdSelectQuery, fmt.Sprintf("%%s%", ir.Title), ir.FeedId).Scan(&itemId)
		if err != nil {
			if err != sql.ErrNoRows {
				log.Fatalf("[Error] [SaveItemToDb] %s", err)
			}
		}
	}

	tx, err := conn.Begin()
	if err != nil {
		log.Fatalf("[Error] [SaveItemToDb] %s", err)
	}
	defer tx.Rollback()

	if itemId > 0 {
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
			active = ?,
			slug = ?
			WHERE
			id = ?`

		stmt, err := tx.Prepare(updateQuery)
		if err != nil {
			log.Fatalf("[Error] [SaveItemToDb] %s", err)
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
			ir.Slug,
			itemId)
		if err != nil {
			log.Fatalf("[Error] [SaveItemToDb] %s", err)
		}

		log.Printf("[Item] [Updates] %v\n", itemId)
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
			active,
			slug)
			VALUES
			(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

		stmt, err := tx.Prepare(insertQuery)
		if err != nil {
			log.Fatalf("[Error] [SaveItemToDb] %s", err)
		}
		defer stmt.Close()

		res, err := stmt.Exec(ir.ExternalId,
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
			ir.Slug)
		if err != nil {
			log.Fatalf("[Error] [SaveItemToDb] %s", err)
		}

		itemId, err = res.LastInsertId()
		if err != nil {
			log.Fatalf("[Error] [SaveItemToDb] %s", err)
		}

		log.Printf("[Item] [Inserts] %v\n", itemId)
	}

	err = tx.Commit()
	if err != nil {
		log.Fatalf("[Error] [SaveItemToDb] %s", err)
	}

	return itemId
}

func DisableFeedItemsFromDb(id int) (rowsAffected int64) {
	log.Printf("[DisableFeedItemsFromDb] %v\n", id)

	tx, err := conn.Begin()
	if err != nil {
		log.Fatalf("[Error] [DisableFeedItemsFromDb] %s", err)
	}
	defer tx.Rollback()

	updateQuery := `UPDATE items SET active = 0
		WHERE
		feed_id = ?`

	stmt, err := tx.Prepare(updateQuery)
	if err != nil {
		log.Fatalf("[Error] [DisableItemsFromDb] %s", err)
	}
	defer stmt.Close()

	res, err := stmt.Exec(id)
	if err != nil {
		log.Fatalf("[Error] [DisableItemsFromDb] %s", err)
	}

	rowCnt, err := res.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}

	err = tx.Commit()
	if err != nil {
		log.Fatalf("[Error] [DisableItemsFromDb] %s", err)
	}

	return rowCnt
}
