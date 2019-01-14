package main

import (
    "os"
    "os/exec"
    "log"
    "strings"
    "database/sql"
    "flag"
    _ "github.com/mattn/go-sqlite3"
    "net"
)

const CREATE_DB_QUERY = "CREATE TABLE clips (clip_id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, clip_content TEXT UNIQUE, clip_insert_date NUMERIC, clip_source TEXT)"
const LIST_QUERY = "SELECT clip_id, replace(clip_content, '\t', '  ') FROM clips ORDER by clip_id DESC"
const SEARCH_QUERY = "SELECT clip_content FROM clips WHERE clip_id = ? LIMIT 1"

const MAX_VALUES = 5

func main() {
    fzfPath := flag.String("fzfpath", "fzf", "Optional path to your fzf binary")
    flag.Parse()

    db := initDatabase()

    fzf := exec.Command(*fzfPath, "--delimiter=\t", "--nth=1", "--with-nth=2", "--no-sort", "--read0")
    stdin, err := fzf.StdinPipe()
    stdout, err := fzf.StdoutPipe()

    fzf.Stderr = os.Stderr
    if err != nil {
        log.Fatal(err)
    }

    rows, err := db.Query(LIST_QUERY)
    if err != nil {
        log.Fatal(err)
    }

    for rows.Next() {
        var clip_id string
        var clip_content string

        err = rows.Scan(&clip_id, &clip_content)
        if err != nil {
            log.Fatal(err)
        }
        stdin.Write([]byte(clip_id + "\t" + clip_content + "\x00"))
    }

    if err := fzf.Start(); nil != err {
        log.Fatal(err)
    }

    buf := make([]byte, 20)
    _, err = stdout.Read(buf)
    if err != nil {
        log.Fatal(err)
    }

    clip_id := strings.Split(string(buf), "\t")[0]

    fzf.Wait()

    //TODO: clipboard subst
    c, err := net.Dial("unix", os.Getenv("HOME") + "/.clitto.sock")
    if err != nil {
        log.Fatal(err)
    }
    defer c.Close()

    _, err = c.Write([]byte(clip_id))
    if err != nil {
        log.Fatal(err)
    }
}

func initDatabase() *sql.DB {
    conn, err := sql.Open("sqlite3", "file:" + os.Getenv("HOME") + "/.clitto.sqlite")
    if err != nil {
        log.Fatal(err)
    }

    var name string

    row := conn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='clips'")

    switch err := row.Scan(&name); err {
    case sql.ErrNoRows:
        _, err := conn.Exec(CREATE_DB_QUERY)
        if err != nil {
            log.Fatal(err)
        }
    case nil:
        log.Println("Tabelle schon da, tschüssing")
    default:
        log.Fatal(err)
    }

    return conn
}
