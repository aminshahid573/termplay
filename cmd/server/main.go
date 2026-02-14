package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
	"context"

	"tictactoe-ssh/internal/config"
	"tictactoe-ssh/internal/db"
	"tictactoe-ssh/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	bm "github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
)

func main() {
	// 1. Init DB
	if err := db.Init(); err != nil {
		log.Fatal("Failed to init Firebase", "err", err)
	}

	// 2. Setup SSH
	s, err := wish.NewServer(
		wish.WithAddress(fmt.Sprintf("%s:%d", config.Host, config.Port)),
		wish.WithHostKeyPath("ssh_host_key"),
		wish.WithMiddleware(
			bm.Middleware(teaHandler),
			logging.Middleware(),
			activeterm.Middleware(),
		),
	)
	if err != nil { log.Fatal(err) }

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)
	
	log.Info("Starting Server", "host", config.Host, "port", config.Port)
	
	go func() {
		if err = s.ListenAndServe(); err != nil && err != ssh.ErrServerClosed {
			log.Error("Listen Error", "err", err)
			done <- nil
		}
	}()

	<-done
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	s.Shutdown(ctx)
}

func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	return ui.InitialModel(s), []tea.ProgramOption{tea.WithAltScreen()}
}
