package game

import "github.com/tsukinoko-kun/gametube/config"

type (
	Game struct {
		Config *config.Game
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

func newGame(slug string) (*Game, error) {
	c, ok := config.FindGame(slug)
	if !ok {
		return nil, ErrInvalidGameSlug
	}

	return &Game{
		Config: c,
	}, nil
}
