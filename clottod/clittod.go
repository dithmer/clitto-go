package main

import (
    "time"
    "os"
    "io"
    "strings"
    "strconv"
    "log"
    "database/sql"
    "net"
    _ "github.com/mattn/go-sqlite3"
    clipboard "github.com/atotto/clipboard"
)

const CREATE_DB_QUERY = "CREATE TABLE clips (clip_id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, clip_content TEXT UNIQUE, clip_insert_date NUMERIC, clip_source TEXT)"
const PASTE_QUERY = "REPLACE INTO clips (clip_content, clip_insert_date, clip_source) VALUES (?, ?, ?)"// content, date, source
const DELETE_OLD_QUERY = "DELETE FROM clips WHERE clip_id IN (SELECT clip_id FROM clips ORDER BY clip_insert_date DESC limit ?, ?)" // min, max
const SEARCH_CONTENT_QUERY = "SELECT clip_content FROM clips WHERE clip_id = ? LIMIT 1" // min, max

const MAX_VALUES = 5

func main() {
    var lastClipboardContent string

    buf := make([]byte, 8)

    if _, err := os.Stat(os.Getenv("HOME") + "/.clitto.lock"); !os.IsNotExist(err) {
        log.Println("lock exists")
        file, err := os.Open(os.Getenv("HOME") + "/.clitto.lock")
        if err != nil {
            log.Fatal(err)
        }
        defer file.Close()

        _, err = file.Read(buf)
        if err != nil {
            log.Fatal(err)
        }
        file.Close()

        pid, err := strconv.ParseInt(strings.Trim(string(buf), "\x00"), 10, 64)
        if err != nil {
            log.Fatal(err)
        }
        log.Println(strconv.FormatInt(pid, 10))

        process, err := os.FindProcess(int(pid))
        if err != nil {
            log.Fatal(err)
        }
        process.Kill()

        log.Println("Removing stale lock file")
        err = os.Remove(os.Getenv("HOME") + "/.clitto.lock")
        if err != nil {
            log.Fatal(err)
        }
    }

    file, err := os.Create(os.Getenv("HOME") + "/.clitto.lock")
    if err != nil {
        log.Fatal(err)
    }

    _, err = file.Write([]byte(strconv.Itoa(os.Getpid())))
    if err != nil {
        log.Fatal(err)
    }
    // TODO: Clean exit, Remove lock file

    db := initDatabase()

    os.Remove(os.Getenv("HOME") + "/.clitto.sock")

    l, err := net.Listen("unix", os.Getenv("HOME") + "/.clitto.sock")
    if err != nil {
        log.Fatal(err)
    }
    go listenForConnection(db, l)

    go cleanUp(db)

    for {
        time.Sleep(time.Second * 1)
        clipboardContent := getClipboardContent("clipboard")
        if clipboardContent == lastClipboardContent {
            continue
        }
        lastClipboardContent = clipboardContent
        storeClipboardContent(db, clipboardContent, "clipboard")
    }
}

func listenForConnection(db *sql.DB, l net.Listener) {
    for {
        fd, err := l.Accept()
        if err != nil {
            log.Fatal(err)
        }

        go handleClittoSockConnection(db, fd)
    }
}

func handleClittoSockConnection(db *sql.DB, c net.Conn) {
    buf := make([]byte, 22)

    for {
        _, err := c.Read(buf)
        if err != nil {
            if err != io.EOF {
                log.Fatal(err)
            }
            break
        }
    }

    log.Println("Coming in: ", string(buf))
    stmt, err := db.Prepare(SEARCH_CONTENT_QUERY)
    if err != nil {
        log.Fatal(err)
    }

    num, err := strconv.ParseInt(strings.Trim(string(buf), "\x00"), 10, 64)
    if err != nil {
        log.Fatal(err)
    }
    row := stmt.QueryRow(num)
    var content string
    err = row.Scan(&content)
    if err != nil {
        log.Println("Havent found something")
    }
    clipboard.WriteAll(content)
}

func cleanUp(db *sql.DB) string {
    for {
        log.Println("Cleaning up...")
        stmt, err := db.Prepare(DELETE_OLD_QUERY)

        _, err = stmt.Exec(MAX_VALUES, MAX_VALUES * 2)
        if err != nil {
            log.Fatal(err)
        }

        time.Sleep(time.Minute)
    }
}

func getClipboardContent(selection string) string {
    clipboardContent, err := clipboard.ReadAll()
    if err != nil {
        log.Fatal(err)
    }

    return clipboardContent
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

func storeClipboardContent(conn *sql.DB, content string, selection string) {
    stmt, err := conn.Prepare(PASTE_QUERY)
    if err != nil {
        log.Fatal(err)
    }

    _, err = stmt.Exec(content, time.Now().Unix(), selection)
    if err != nil {
        log.Fatal(err)
    }
}
