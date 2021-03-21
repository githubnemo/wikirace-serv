package main

import (
	"testing"
	"wikirace-serv/wikis"
)

func TestGameHasOwnerAsPlayer(t *testing.T) {
	hostPlayerName := "the player"
	gameWiki := &wikis.Wiki{}
	game := NewGame(hostPlayerName, gameWiki, nil)

	if game.Host != hostPlayerName {
		t.Errorf("game.Host is %s and should be %s", game.Host, hostPlayerName)
	}

	if len(game.Players) != 1 {
		t.Errorf("Expected player list to be at least 1, got %d", len(game.Players))
	}

	if game.Players[0].Name != hostPlayerName {
		t.Errorf("Expected first player to be %s but got %s",
			hostPlayerName, game.Players[0].Name)
	}
}
