package main

type Game struct {
	// Cache for the game hash
	hash string

	// Name of the player who initiated the game
	Host string

	// All players including the host
	PlayerHashes []string

	// The winner of the game. Empty if the game is not finished yet
	Winner string

	// The path the winner took to the goal. Empty if the game is not finished
	WinnerPath []string

	// Name of the start and goal article of this game
	Start string
	Goal string
}

func NewGame(hostingPlayerName string) *Game {
	return &Game{
		Host: hostingPlayerName,
		PlayerHashes: []string{hostingPlayerName},
	}
}

func (g *Game) Hash() string {
	if len(g.hash) == 0 {
		g.hash = computeGameHash(g.Host)
	}

	return g.hash
}

func (g *Game) AddPlayer(name string) {
	g.PlayerHashes = append(g.PlayerHashes, name)
}

func (g *Game) Save() error {
	return gameStore.PutMarshal(g.Hash(), g)
}


// Compute the game hash using the hosting player's name
func computeGameHash(playerName string) string {
	// TODO: actual hash
	return playerName + "itsahashdealwithit"
}

// The key has to be present.
func getGameByHash(hash string) (*Game, error) {
	var game Game

	err := gameStore.GetMarshal(hash, &game)

	if err != nil {
		return nil, err
	}

	return &game, nil
}

// Searches for a game hosted by the player with the given name.
// Returns nil if no game was found.
func findRunningGame(playerName string) *Game {
	gameHash := computeGameHash(playerName)

	if !gameStore.Contains(gameHash) {
		return nil
	}

	game, err := getGameByHash(gameHash)

	if err != nil {
		// This should not happen and indicates bad data
		// as we clearly have a document with the hash.
		panic(err)
	}

	return game
}

