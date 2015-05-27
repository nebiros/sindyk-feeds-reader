package reader

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"fmt"
	"log"
)

const (
	// "user:password@tcp(localhost:3306)/dbname??charset=utf8"
	DefaultDsnFormat = "%s:%s@tcp(%s:%s)/%s?charset=%s"
	DefaultDriver = "mysql"
)

var DB *sql.DB;

func buildDSN(p Params) string {
	dsn := fmt.Sprintf(DefaultDsnFormat,
		p.Username,
		p.Password,
		p.Address,
		p.Port,
		p.Database,
		p.Charset);

	return dsn
}

func OpenDb(p Params) {
	var err error

	dsn := buildDSN(p)
	DB, err = sql.Open(DefaultDriver, dsn)
	if err != nil {
		log.Fatal(err)
	}

	err = DB.Ping()
	if err != nil {
		log.Fatal(err)
	}
}

func Feeds() (feeds []Feed) {
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
		f := Feed{}

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