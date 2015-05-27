package main

import (
	"flag"
	"fmt"
	"os"
	"github.com/nebiros/sindyk-feeds-reader/lib/reader"
)

func main() {
	// set cli flags.
	dbaddress := flag.String("dbaddress", "127.0.0.1", "Database Address. Defaults to \"127.0.0.1\"");
	dbusername := flag.String("dbusername", "root", "Database Username. Defaults to \"root\"");
	dbpassword := flag.String("dbpassword", "", "Database Password. Defaults to \"\"");
	dbname := flag.String("dbname", "", "Database Name. Required");
	dbport := flag.String("dbport", "3306", "Database Port Number. Defaults to \"3306\"");
	dbcharset := flag.String("dbcharset", "utf8", "Database Charset. Defaults to \"utf8\"");
	flag.Parse();

	if len(*dbname) <= 0 {
		fmt.Println("Database Name. Required");
		flag.Usage()
		os.Exit(1)
	}

	// start parsing.
	params := reader.Params{Address: *dbaddress,
		Username: *dbusername,
		Password: *dbpassword,
		Database: *dbname,
		Port: *dbport,
		Charset: *dbcharset}
	reader.Start(params)
}