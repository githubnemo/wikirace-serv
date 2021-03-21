package main

import (
	"testing"
	"wikirace-serv/wikis"
)

func TestGameHasOwnerAsPlayer(t *testing.T) {
	hostPlayerName := "player 1"
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


func simpleTwoPlayerGame() *Game {
	hostPlayerName := "player 1"
	gameWiki := &wikis.Wiki{}
	game := NewGame(hostPlayerName, gameWiki, nil)

	game.Start = "start page"
	game.Goal = "goal page"

	playerName1 := hostPlayerName
	playerName2 := "player 2"

	game.AddPlayer(playerName2)

	player1 := game.GetPlayer(playerName1)
	player2 := game.GetPlayer(playerName2)

	player1.Visited(game.Start)
	player2.Visited(game.Start)

	return game
}

func TestSimpleGame(t *testing.T) {
	game := simpleTwoPlayerGame()

	if len(game.Players) != 2 {
		t.Errorf("Unexpected number of players %d", len(game.Players))
	}
}

func TestNobodyWonSimpleGameAtStart(t *testing.T) {
	game := simpleTwoPlayerGame()

	player1 := &game.Players[0]
	player2 := &game.Players[1]

	isWinner, isTempWinner := game.EvaluateWinner(player1)
	if isWinner || isTempWinner {
		t.Errorf("player1: winner: %t, temporary: %t, expected both false", isWinner, isTempWinner)
	}

	isWinner, isTempWinner = game.EvaluateWinner(player2)
	if isWinner || isTempWinner {
		t.Errorf("player2: winner: %t, temporary: %t, expected both false", isWinner, isTempWinner)
	}
}

func TestWinnerSimple(t *testing.T) {
	game := simpleTwoPlayerGame()

	player1 := &game.Players[0]
	player2 := &game.Players[1]

	// player1 screws up, player2 finds the goal page
	// there is no way player1 can catch up, player2 is the winner.
	player1.Visited("other page")
	player2.Visited(game.Goal)

	isWinner, isTempWinner := game.EvaluateWinner(player1)
	if isWinner || isTempWinner {
		t.Errorf("player1: winner: %t, temporary: %t, expected both false", isWinner, isTempWinner)
	}

	isWinner, isTempWinner = game.EvaluateWinner(player2)
	if !isWinner {
		t.Errorf("player2: winner: %t, temporary: %t, expected both true", isWinner, isTempWinner)
	}
}
