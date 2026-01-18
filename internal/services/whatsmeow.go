package services

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"chatbot-whatsmeow/internal/websocket"

	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

var reconnecting atomic.Bool

type WhatsMeowService struct {
	Client   *whatsmeow.Client
	Store    *store.Device
	Container *sqlstore.Container
}

func NewWhatsMeowService(ctx context.Context, dbLog waLog.Logger) (*WhatsMeowService, error) {
	container, err := sqlstore.New(ctx, "sqlite3", "file:examplestore.db?_foreign_keys=on", dbLog)
	if err != nil {
		return nil, err
	}
	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, err
	}
	clientLog := waLog.Stdout("Client", "INFO", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)
	client.ManualHistorySyncDownload = true

	service := &WhatsMeowService{
		Client:    client,
		Store:     deviceStore,
		Container: container,
	}

	client.AddEventHandler(service.eventHandler())

	return service, nil
}

func (s *WhatsMeowService) eventHandler() func(evt interface{}) {
	return func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Message:
			fmt.Printf("[MSG] From %s: %s\n", v.Info.Sender.User, v.Message.GetConversation())
		case *events.Disconnected:
			fmt.Printf("[Client WARN] Disconnected: %+v\n", v)
			websocket.Broadcast([]byte(`{"type": "status", "status": "disconnected"}`))
			s.scheduleReconnect()
		case *events.Connected:
			fmt.Printf("[Client INFO] Connected\n")
			websocket.Broadcast([]byte(`{"type": "status", "status": "connected"}`))
		case *events.QR:
			fmt.Printf("[Client INFO] QR code: %s\n", v.Codes[0])
			websocket.Broadcast([]byte(fmt.Sprintf(`{"type": "qr", "code": "%s"}`, v.Codes[0])))
		case *events.LoggedOut:
			fmt.Printf("[Client WARN] Logged out from another device, deleting session and relogging...\n")
			websocket.Broadcast([]byte(`{"type": "status", "status": "logged_out"}`))
			ctx := context.Background()
			err := s.Container.DeleteDevice(ctx, s.Store)
			if err != nil {
				fmt.Printf("[Client ERROR] Failed to delete device: %v\n", err)
			}
			s.Client.Disconnect()
			s.Client.Store.ID = nil
			go s.handleReconnect()
		}
	}
}

func (s *WhatsMeowService) scheduleReconnect() {
	if !reconnecting.CompareAndSwap(false, true) {
		return
	}

	go func() {
		defer reconnecting.Store(false)
		backoff := time.Second
		for {
			err := s.Client.Connect()
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

func (s *WhatsMeowService) handleReconnect() {
	qrChan, _ := s.Client.GetQRChannel(context.Background())
	err := s.Client.Connect()
	if err != nil {
		fmt.Printf("[Client ERROR] Failed to reconnect: %v\n", err)
		return
	}
	for evt := range qrChan {
		if evt.Event == "code" {
			fmt.Println("Scan the QR code above with WhatsApp")
			// Broadcast will be handled in websocket package
		} else {
			fmt.Println("Login event:", evt.Event)
		}
	}
}

func (s *WhatsMeowService) Connect() error {
	if s.Client.Store.ID == nil {
		// New login
		qrChan, _ := s.Client.GetQRChannel(context.Background())
		err := s.Client.Connect()
		if err != nil {
			return err
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				fmt.Println("Scan the QR code above with WhatsApp")
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		// Already logged in
		return s.Client.Connect()
	}
	return nil
}

func (s *WhatsMeowService) Disconnect() {
	s.Client.Disconnect()
}

func (s *WhatsMeowService) Logout(ctx context.Context) error {
	return s.Client.Logout(ctx)
}

func (s *WhatsMeowService) IsConnected() bool {
	return s.Client.IsConnected()
}

func (s *WhatsMeowService) IsLoggedIn() bool {
	return s.Client.IsLoggedIn()
}