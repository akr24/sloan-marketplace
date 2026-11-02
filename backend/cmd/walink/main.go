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
	"go.mau.fi/whatsmeow/types/events"

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

	// After pairing, WhatsApp restarts the socket and the client logs in for
	// the first time; disconnecting before that finishes gets the new device
	// logged out again, so wait for it.
	connected := make(chan struct{}, 1)
	client.AddEventHandler(func(evt any) {
		if _, ok := evt.(*events.Connected); ok {
			select {
			case connected <- struct{}{}:
			default:
			}
		}
	})

	linkCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	err = whatsappweb.Link(linkCtx, client, phone, func(code string) {
		fmt.Printf("\n  Pairing code for +%s:  %s\n\n", phone, code)
		fmt.Println("  On that phone: WhatsApp → Settings → Linked devices → Link a device →")
		fmt.Println("  \"Link with phone number instead\", then enter the code. You have about 2 minutes.")
		fmt.Println()
	})
	if err == nil {
		select {
		case <-connected:
			time.Sleep(5 * time.Second) // let the first sync settle
		case <-time.After(time.Minute):
			err = fmt.Errorf("whatsapp-web: paired, but the first login never finished")
		}
	}
	client.Disconnect()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Linked. Start the server with WHATSAPP_WEB_ENABLED=true and add this number to your groups.")
}
