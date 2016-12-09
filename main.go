package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os/user"

	_ "github.com/mattn/go-sqlite3"
)

var (
	phone = flag.String("phone", "", "The iMessage ID (phone/email) to search against")
	text  = flag.String("q", "", "the text to search for")

	fromMap = map[int]string{
		0: "Them",
		1: "You",
	}
)

type line struct {
	IsMe    int
	Date    string
	Message string
}

func main() {
	flag.Parse()

	usr, err := user.Current()
	if err != nil {
		log.Fatal("could not get current user")
	}

	db, err := sql.Open("sqlite3", usr.HomeDir+"/Library/Messages/chat.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	query := fmt.Sprintf(`
        SELECT  
            is_from_me, 
            datetime(date + strftime('%%s', '2001-01-01'), 'unixepoch', 'localtime'),
            trim(text) 
        FROM    message 
        WHERE   
            text LIKE '%%%s%%' --case-insensitive
            AND     handle_id=(
                SELECT  handle_id
                FROM    chat_handle_join 
                WHERE   chat_id=(
                    SELECT  ROWID 
                    FROM    chat 
                    WHERE   guid='iMessage;-;%s'
                )
            )
            AND NOT text LIKE '%%Digital Touch Message%%'`, *text, *phone)

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
