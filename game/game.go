package game

import (
	"context"
	"errors"

	"github.com/tsukinoko-kun/gametube/config"
)

type (
	Game struct {
		Config    *config.Game
		Container *Container
	}
)

var (
	activeGames = make(map[string]*Game)
)

func (g *Game) Stop() {
}

func (g *Game) Title() string {
	return g.Config.Title
}

func newGame(ctx context.Context, sessionId, slug string) (*Game, error) {
	c, ok := config.FindGame(slug)
	if !ok {
		return nil, ErrInvalidGameSlug
	}

	container, err := NewContainer(ctx, sessionId, c)
	if err != nil {
		return nil, errors.Join(errors.New("failed to create container"), err)
	}
	defer container.Close()

	return &Game{
		Config:    c,
		Container: container,
	}, nil
}

func GetGame(sessionId string) (*Game, bool) {
	game, ok := activeGames[sessionId]
	return game, ok
}
