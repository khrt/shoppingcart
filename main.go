package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	var (
		dsn  = flag.String("dsn", "file:./testdata/db.sqlite3?cache=shared&_loc=UTC&mode=rw", "DSN")
		addr = flag.String("addr", ":5000", "Address to bind HTTP server")
	)
	flag.Parse()

	db, err := sql.Open("sqlite3", *dsn)
	if err != nil {
		log.Fatal("db:", db)
	}

	st := &SQLite3{db: db}
	sc := &ShoppingCart{storage: st}

	s := &http.Server{
		Addr:    *addr,
		Handler: NewAPIv1(sc),
	}

	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint

		// We received an interrupt signal, shut down.
		if err := s.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout:
			log.Println("http shutdown:", err)
		}
		close(idleConnsClosed)
	}()

	log.Printf("Listening on %s...", *addr)
	if err := s.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal("http:", err)
	}

	<-idleConnsClosed
}
