package main

import (
	"bytes"
	"io"
	"poker/window"
	"strings"
	"time"

	"os"

	"github.com/whomever000/poker-client-pokerstars/vision"
	poker "github.com/whomever000/poker-common"

	log "github.com/Sirupsen/logrus"
)

// This file contains utility functions.

////////////////////////////////////////////////////////////////////////////////
// Hand header
////////////////////////////////////////////////////////////////////////////////

// client returns the name of the client.
func client() string {
	return "PokerStars"
}

// table returns the details of the table.
func table() poker.Table {

	var table poker.Table

	// Get window name.
	log.Debugf("window %v", window.Get())
	name, err := window.Get().Name()
	if err != nil {
		log.Warnf("failed to get window name. %v", err)
	}

	strs := strings.Split(name, " - ")
	if len(strs) < 3 {
		log.Errorf("expected three or more substrings in window name. Got: %v", name)
		return table
	}

	table.Name = strs[0]
	table.Stakes, err = poker.ParseStakes(strs[1])
	if err != nil {
		log.Errorf("failed to parse table stakes. %v", err)
	}
	// TODO: allow other table sizes.
	table.Size = 6
	table.Game, err = poker.ParseGame(strs[2])
	if err != nil {
		log.Errorf("failed to parse game. %v", err)
	}

	return table
}

// handID returns current hand ID.
func handID() int {
	return os.Getpid()
}

// date returns the time the hand was started.
func date() poker.Date {
	return poker.Date(time.Now())
}

// button returns the current button position.
func button() poker.PlayerPosition {
	return vision.ButtonPosition(img)
}

// smallBlind returns the small blind position.
func smallBlind() poker.PlayerPosition {
	return nextActivePlayer(h.Button)
}

// bigBlind returns the big blind position.
func bigBlind() poker.PlayerPosition {
	return nextActivePlayer(h.SmallBlind)
}

// thisPlayer returns information about 'me'.
func thisPlayer() *poker.PlayerCards {
	cards, err := vision.PocketCards(img)
	if err != nil {
		log.Printf("error: Failed to get pocket cards. %v", err)
		return nil
	}

	return &poker.PlayerCards{
		Position: 4,
		Cards:    cards,
	}
}

// players returns information about all players.
func players() []poker.Player {

	sync := make(chan bool, 6)

	players := make([]poker.Player, 6)
	for i := 0; i < 6; i++ {
		index := i
		go func() {
			name, _ := vision.PlayerName(img, poker.PlayerPosition(index+1))
			stack, err := vision.PlayerStack(img, poker.PlayerPosition(index+1))
			if err != nil {
				panic(err)
			}
			log.Info(name, stack)

			players[index] = poker.Player{Name: name, Stack: stack}
			playerStacks[index] = stack
			sync <- true
		}()
	}

	for i := 0; i < 6; i++ {
		<-sync
	}

	return players
}

////////////////////////////////////////////////////////////////////////////////
// Custom file loader
////////////////////////////////////////////////////////////////////////////////

// fileLoader is used for loading files through bindata library.
type fileLoader struct{}

func (l *fileLoader) Load(fileName string) io.Reader {
	// Remove relative pefix (bindata cannot handle)
	if strings.HasPrefix(fileName, "./") {
		fileName = fileName[2:]
	}

	// Prepend 'res/' folder
	fileName = "res/" + fileName

	// Load file through bindata.
	data, err := Asset(fileName)
	if err != nil {
		log.Printf("error: Failed to load file %v", fileName)
		return nil
	}

	return bytes.NewReader(data)
}
