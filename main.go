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
		0: "You:\t",
		1: "Them:\t",
	}
)

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
        SELECT  is_from_me, text 
        FROM    message 
        WHERE   text LIKE '%%%s%%' --case-insensitive
        AND     handle_id=(
            SELECT  handle_id
            FROM    chat_handle_join 
            WHERE   chat_id=(
                SELECT  ROWID 
                FROM    chat 
                WHERE   guid='iMessage;-;%s'
            )
        )`, *text, *phone)

	rows, err := db.Query(query)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var isMe int
		var msg string
		err = rows.Scan(&isMe, &msg)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(fromMap[isMe], msg)
	}
}
