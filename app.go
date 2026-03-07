package main

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"clickwheel/internal/audiobookshelf"
	"clickwheel/internal/config"
	"clickwheel/internal/ipod"
	"clickwheel/internal/navidrome"
	"clickwheel/internal/sync"
)

type App struct {
	ctx        context.Context
	cfg        *config.Config
	navClient  *navidrome.Client
	absClient  *audiobookshelf.Client
	syncEngine *sync.Engine
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	cfg, err := config.Load()
	if err != nil {
		cfg = config.Default()
	}
	a.cfg = cfg
}

func (a *App) shutdown(ctx context.Context) {
	if a.cfg != nil {
		_ = a.cfg.Save()
	}
}

func (a *App) GetConfig() *config.Config {
	return a.cfg
}

func (a *App) SaveNavidromeConfig(serverURL, username, password string) error {
	a.cfg.Navidrome.ServerURL = serverURL
	a.cfg.Navidrome.Username = username
	a.cfg.Navidrome.Password = password
	a.navClient = navidrome.NewClient(serverURL, username, password)
	return a.cfg.Save()
}

func (a *App) SaveABSConfig(serverURL, token string) error {
	a.cfg.ABS.ServerURL = serverURL
	a.cfg.ABS.Token = token
	a.absClient = audiobookshelf.NewClient(serverURL, token)
	return a.cfg.Save()
}

func (a *App) TestNavidromeConnection() error {
	if a.navClient == nil {
		a.navClient = navidrome.NewClient(
			a.cfg.Navidrome.ServerURL,
			a.cfg.Navidrome.Username,
			a.cfg.Navidrome.Password,
		)
	}
	return a.navClient.Ping()
}

func (a *App) TestABSConnection() error {
	if a.absClient == nil {
		a.absClient = audiobookshelf.NewClient(a.cfg.ABS.ServerURL, a.cfg.ABS.Token)
	}
	return a.absClient.Ping()
}

func (a *App) DetectIPod() (*ipod.DeviceInfo, error) {
	return ipod.Detect()
}

func (a *App) GetNavidromePlaylists() ([]navidrome.Playlist, error) {
	if a.navClient == nil {
		return nil, nil
	}
	return a.navClient.GetPlaylists()
}

func (a *App) GetNavidromeAlbums() ([]navidrome.Album, error) {
	if a.navClient == nil {
		return nil, nil
	}
	return a.navClient.GetAlbums(0, 500)
}

func (a *App) GetABSLibraries() ([]audiobookshelf.Library, error) {
	if a.absClient == nil {
		return nil, nil
	}
	return a.absClient.GetLibraries()
}

func (a *App) GetABSBooks(libraryID string) ([]audiobookshelf.Book, error) {
	if a.absClient == nil {
		return nil, nil
	}
	return a.absClient.GetBooks(libraryID)
}

func (a *App) GetExclusions() config.Exclusions {
	return a.cfg.Exclusions
}

func (a *App) SetExclusions(exclusions config.Exclusions) error {
	a.cfg.Exclusions = exclusions
	return a.cfg.Save()
}

func (a *App) StartSync() error {
	if a.syncEngine == nil {
		a.syncEngine = sync.NewEngine(a.cfg, a.navClient, a.absClient)
	}
	go func() {
		err := a.syncEngine.Run(a.ctx, func(progress sync.Progress) {
			runtime.EventsEmit(a.ctx, "sync:progress", progress)
		})
		if err != nil {
			runtime.EventsEmit(a.ctx, "sync:error", err.Error())
			return
		}
		runtime.EventsEmit(a.ctx, "sync:done", nil)
	}()
	return nil
}
