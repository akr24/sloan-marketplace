// Command walink links the bot's WhatsApp account to this server, once.
//
// It asks WhatsApp for an 8-character pairing code for WHATSAPP_WEB_PHONE and
// prints it; enter it on that phone under Settings → Linked devices → Link a
// device → "Link with phone number instead". The session is saved in
// Postgres, after which the server's group bot can connect on its own.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/jackc/pgx/v5/stdlib"

	"github.com/akr24/sloan-marketplace/backend/internal/adapter/postgres"
	"github.com/akr24/sloan-marketplace/backend/internal/adapter/whatsappweb"
	"github.com/akr24/sloan-marketplace/backend/internal/usecase"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	phone, err := usecase.NormalisePhone(os.Getenv("WHATSAPP_WEB_PHONE"))
	if err != nil {
		log.Fatal("set WHATSAPP_WEB_PHONE to the bot's number, e.g. +1 872 266 7930")
	}

	pool, err := postgres.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()
	db := stdlib.OpenDBFromPool(pool)

	client, err := whatsappweb.Open(ctx, db)
	if err != nil {
		log.Fatal(err)
	}
	if whatsappweb.Linked(client) {
		fmt.Printf("Already linked as +%s. To relink, remove the device on the phone (Linked devices), start the server once so it notices, then run this again.\n", client.Store.ID.User)
		return
	}

	linkCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	err = whatsappweb.Link(linkCtx, client, phone, func(code string) {
		fmt.Printf("\n  Pairing code for +%s:  %s-%s\n\n", phone, code[:4], code[4:])
		fmt.Println("  On that phone: WhatsApp → Settings → Linked devices → Link a device →")
		fmt.Println("  \"Link with phone number instead\", then enter the code. You have about 2 minutes.")
		fmt.Println()
	})
	client.Disconnect()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Linked. Start the server with WHATSAPP_WEB_ENABLED=true and add this number to your groups.")
}
