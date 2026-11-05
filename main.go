package main

import (
	"context"
	"log"
	"os"

	_ "github.com/gogpu/gg/gpu"
	"github.com/gogpu/gogpu"
	"github.com/gogpu/ui/app"
	"github.com/gogpu/ui/desktop"

	"chordly/appui"
	"chordly/internal/ytm"
	"chordly/router"
)

func main() {
	routes := router.New(
		&router.Route{
			Segment: "",
			Name:    "home",
			Page:    appui.HomePage,
		},
		&router.Route{
			Segment: "explore",
			Name:    "explore",
			Page:    appui.HomePage,
		},
		&router.Route{
			Segment: "library",
			Name:    "library",
			Page:    appui.HomePage,
		},
		&router.Route{
			Segment: "search",
			Name:    "search",
			Page:    appui.SearchPage,
		},
		&router.Route{
			Segment: "queue",
			Name:    "queue",
			Page:    appui.QueuePage,
		},
		&router.Route{
			Segment: "lyrics",
			Name:    "lyrics",
			Page:    appui.LyricsPage,
		},
		&router.Route{
			Segment: "settings",
			Name:    "settings",
			Page:    appui.SettingsPage,
		},
		&router.Route{
			Segment: "*",
			Name:    "not-found",
			Page:    appui.NotFoundPage,
		},
	)

	if len(os.Args) > 1 {
		routes.Replace(os.Args[1])
	}

	// Trigger live YouTube Music home feed synchronization
	ytm.DefaultService.SyncHomeData(context.Background())

	gogpuApp := gogpu.NewApp(gogpu.DefaultConfig().
		WithTitle("Chordly - YouTube Music").
		WithSize(1280, 820))

	uiApp := app.New(
		app.WithWindowProvider(gogpuApp),
		app.WithPlatformProvider(gogpuApp),
		app.WithEventSource(gogpuApp.EventSource()),
	)

	uiApp.SetRoot(appui.NewShell(routes))

	if err := desktop.Run(gogpuApp, uiApp); err != nil {
		log.Fatal(err)
	}
}
