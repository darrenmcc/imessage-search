package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/user"

	"strings"

	"github.com/fatih/color"
	_ "github.com/mattn/go-sqlite3"
)

const defaultDBLocation = "/Library/Messages/chat.db"

var (
	id         = flag.String("id", "", "The iMessage ID (phone/email) to search against")
	text       = flag.String("q", "", "the text to search for")
	dbLocation = flag.String("db", "", "the location of your chat.db file, if other than ~/Library/Messages/chat.db")

	fileDump         = flag.Bool("file-dump", false, "dump all shared files")
	fileDumpLocation = flag.String("dir", ".", "the location to dump files")

	fromMap = map[int]string{
		0: color.RedString("Them"),
		1: color.BlueString("You"),
	}
)

type line struct {
	IsMe    int
	Date    string
	Message string
}

func main() {
	flag.Parse()

	if *id == "" {
		log.Fatal("iMessage ID (+12345675555 or email) required")
	}

	usr, err := user.Current()
	if err != nil {
		log.Fatal("could not get current user")
	}

	var dbLoc string
	if *dbLocation == "" {
		dbLoc = usr.HomeDir + defaultDBLocation
	}

	db, err := sql.Open("sqlite3", dbLoc)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if *fileDump {
		getFiles(db, usr.HomeDir)
		return
	}

	getMessages(db)

}

func getMessages(db *sql.DB) {
	query := fmt.Sprintf(`
        SELECT  
            is_from_me, 
            datetime(date + strftime('%%s', '2001-01-01'), 'unixepoch', 'localtime'),
            trim(text) 
        FROM    message 
        WHERE   text LIKE '%%%s%%' --case-insensitive
        AND     handle_id=(
            SELECT  handle_id
            FROM    chat_handle_join 
            WHERE   chat_id=(
                SELECT  ROWID 
                FROM    chat 
                WHERE   chat_identifier='%s'
            )
        )
        AND NOT text LIKE '%%Digital Touch Message%%'`, *text, *id)

	rows, err := db.Query(query)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var isMe int
		var msg, date string
		err = rows.Scan(&isMe, &date, &msg)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s...%s:\t%s\n", date, fromMap[isMe], msg)
	}
}

func getFiles(db *sql.DB, homeDir string) {
	query := fmt.Sprintf(`
        SELECT  filename 
        FROM    attachment 
        WHERE   rowid IN (
            SELECT  attachment_id 
            FROM    message_attachment_join 
            WHERE   message_id in (
                SELECT  rowid 
                FROM    message 
                WHERE   cache_has_attachments=1 
                AND     handle_id=(
                    SELECT  handle_id 
                    FROM    chat_handle_join 
                    WHERE   chat_id=(
                        SELECT  ROWID 
                        FROM    chat 
                        WHERE   chat_identifier='%s'
                    )
                )
            )
        )`, *id)

	rows, err := db.Query(query)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var file string
		err = rows.Scan(&file)
		if err != nil {
			log.Fatal(err)
		}
		src := strings.Replace(file, "~", homeDir, 1)
		tokens := strings.Split(src, "/")
		dst := *fileDumpLocation + "/" + tokens[len(tokens)-1]
		_, err := copyFile(src, dst)
		if err != nil {
			log.Print("could not copy file: " + err.Error())
		}
	}
}

func copyFile(src, dst string) (int64, error) {
	src_file, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer src_file.Close()

	src_file_stat, err := src_file.Stat()
	if err != nil {
		return 0, err
	}

	if !src_file_stat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	dst_file, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer dst_file.Close()
	return io.Copy(dst_file, src_file)
}
