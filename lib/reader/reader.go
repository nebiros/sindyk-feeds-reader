package reader

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"html"
	"log"
	"net/url"
	"runtime"
	"strings"
)

const (
	// "user:password@tcp(localhost:3306)/dbname??charset=utf8"
	DbDefaultDsnFormat = "%s:%s@tcp(%s:%s)/%s?charset=%s"
	DbDefaultDriver    = "mysql"
)

var (
	// db connection.
	conn *sql.DB
)

type Params struct {
	Address, Username, Password, Database, Port, Charset string
}

type FeedRow struct {
	Id     int
	Url    string
	Active int
}

type ItemRow struct {
	Id           int
	ExternalId   int
	FeedId       int
	Title        string
	Description  string
	Link         string
	PubDate      string
	Content      string
	Creator      string
	ImageUrl     string
	Active       int
	DisplayOrder int
	Subject      string
	Category     string
	Hour         string
	Related      string
	Slug         string
}

type FetchedFeed struct {
	FeedId int
	Rss    *Rss
}

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	log.SetFlags(log.LstdFlags | log.Lshortfile)
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
		p.Charset)

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
		AND sections.other != ?
		WHERE
		feeds.active = ?
		ORDER BY feeds.id ASC`

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
	fetchedFeedsChan, errorsChan := make(chan *FetchedFeed), make(chan error)

	for _, f := range af {
		go func(f *FeedRow) {
			log.Printf("[Process] %s\n", f.Url)

			DisableFeedItemsFromDb(f.Id)

			r, err := FetchRss(f.Url)
			if err != nil {
				errorsChan <- err
			} else {
				fetchedFeedsChan <- &FetchedFeed{FeedId: f.Id, Rss: r}
			}
		}(f)
	}

	for _, f := range af {
		select {
		case ff := <-fetchedFeedsChan:
			if ff.Rss == nil {
				continue
			}

			Marshal(ff.FeedId, ff.Rss.RssItemList)
		case err := <-errorsChan:
			log.Printf("[Error] [Process] [%s] %s\n", f.Url, err)
		}
	}
}

func Marshal(id int, il []*RssItem) {
	for _, i := range il {
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
			log.Printf("[Error] [Process] %s\n", err)
		}

		slug := u.Path
		if string([]rune(slug)[0]) == "/" {
			slug = slug[1:len(slug)]
		}

		log.Printf("[Item] [Start] %s\n", link)

		ir := &ItemRow{ExternalId: i.Id,
			FeedId:       id,
			Title:        strings.TrimSpace(i.Title),
			Description:  html.EscapeString(i.Description),
			Link:         link,
			PubDate:      strings.TrimSpace(i.PubDate),
			Content:      html.EscapeString(content),
			Creator:      html.EscapeString(creator),
			ImageUrl:     imageUrl,
			Active:       1,
			DisplayOrder: i.Order,
			Category:     strings.TrimSpace(i.Category),
			Subject:      subject,
			Hour:         strings.TrimSpace(i.Hour),
			Related:      strings.TrimSpace(i.Related),
			Slug:         slug}
		SaveItemToDb(ir)
	}
}

func SaveItemToDb(ir *ItemRow) (id int64) {
	var (
		itemIdSelectQuery string
		itemId            int64
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

		err := conn.QueryRow(itemIdSelectQuery, fmt.Sprintf("%%%s%%", ir.Title), ir.FeedId).Scan(&itemId)
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

	var (
		totalItems   int64
		disableItems []string
	)

	totalItemsQuery := `SELECT COUNT(id)
		FROM items
		WHERE feed_id = ?`
	err := conn.QueryRow(totalItemsQuery, id).Scan(&totalItems)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Fatalf("[Error] [DisableFeedItemsFromDb] %s", err)
		}
	}

	itemsDisableQuery := `SELECT id FROM items
		WHERE feed_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET 10`
	rows, err := conn.Query(itemsDisableQuery, id, totalItems)
	if err != nil {
		log.Fatalf("[Error] [DisableFeedItemsFromDb] %s", err)
	}
	defer rows.Close()

	for rows.Next() {
		var itemId string

		err := rows.Scan(&itemId)
		if err != nil {
			log.Fatalf("[Error] [DisableFeedItemsFromDb] %s", err)
		}

		disableItems = append(disableItems, itemId)
	}
	err = rows.Err()
	if err != nil {
		log.Fatalf("[Error] [DisableFeedItemsFromDb] %s", err)
	}

	tx, err := conn.Begin()
	if err != nil {
		log.Fatalf("[Error] [DisableFeedItemsFromDb] %s", err)
	}
	defer tx.Rollback()

	updateQuery := `UPDATE items SET active = 0
		WHERE
		id IN (?)`
	stmt, err := tx.Prepare(updateQuery)
	if err != nil {
		log.Fatalf("[Error] [DisableItemsFromDb] %s", err)
	}
	defer stmt.Close()

	res, err := stmt.Exec(strings.Join(disableItems, ", "))
	if err != nil {
		log.Fatalf("[Error] [DisableItemsFromDb] %s", err)
	}

	count, err := res.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}

	err = tx.Commit()
	if err != nil {
		log.Fatalf("[Error] [DisableItemsFromDb] %s", err)
	}

	return count
}
