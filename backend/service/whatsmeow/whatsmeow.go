package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

var reconnecting atomic.Bool

func scheduleReconnect(client *whatsmeow.Client) {
	if !reconnecting.CompareAndSwap(false, true) {
		return
	}

	go func() {
		defer reconnecting.Store(false)
		backoff := time.Second
		for {
			err := client.Connect()
			if err == nil {
				fmt.Println("[Client INFO] Reconnected successfully")
				return
			}
			fmt.Printf("[Client WARN] Reconnect failed: %v\n", err)
			time.Sleep(backoff)
			if backoff < 30*time.Second {
				backoff *= 2
			}
		}
	}()
}

func eventHandler(container *sqlstore.Container, deviceStore *store.Device, client *whatsmeow.Client) func(evt interface{}) {
	return func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Message:
			fmt.Printf("[MSG] From %s: %s\n", v.Info.Sender.User, v.Message.GetConversation())
		case *events.Disconnected:
			fmt.Printf("[Client WARN] Disconnected: %+v\n", v)
			scheduleReconnect(client)
		case *events.LoggedOut:
			fmt.Printf("[Client WARN] Logged out from another device, deleting session and relogging...\n")
			ctx := context.Background()
			err := container.DeleteDevice(ctx, deviceStore)
			if err != nil {
				fmt.Printf("[Client ERROR] Failed to delete device: %v\n", err)
			}
			client.Disconnect()
			client.Store.ID = nil
			go func() {
				qrChan, _ := client.GetQRChannel(context.Background())
				err = client.Connect()
				if err != nil {
					fmt.Printf("[Client ERROR] Failed to reconnect: %v\n", err)
					return
				}
				for evt := range qrChan {
					if evt.Event == "code" {
						fmt.Println("Scan the QR code above with WhatsApp")
						qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
					} else {
						fmt.Println("Login event:", evt.Event)
					}
				}
			}()
		}
	}
}

func main() {

	dbLog := waLog.Stdout("Database", "INFO", true)
	ctx := context.Background()
	container, err := sqlstore.New(ctx, "sqlite3", "file:examplestore.db?_foreign_keys=on", dbLog)
	if err != nil {
		panic(err)
	}
	// If you want multiple sessions, remember their JIDs and use .GetDevice(jid) or .GetAllDevices() instead.
	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		panic(err)
	}
	clientLog := waLog.Stdout("Client", "INFO", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)
	client.ManualHistorySyncDownload = true
	client.AddEventHandler(eventHandler(container, deviceStore, client))

	if client.Store.ID == nil {
		// No ID stored, new login
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			panic(err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				// Render the QR code here
				fmt.Println("Scan the QR code above with WhatsApp")
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				// or just manually `echo 2@... | qrencode -t ansiutf8` in a terminal
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		// Already logged in, just connect
		err = client.Connect()
		if err != nil {
			panic(err)
		}
	}

	// Listen to Ctrl+C (you can also do something else that prevents the program from exiting)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	client.Disconnect()
}
