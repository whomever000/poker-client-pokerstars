package main

// // No C code needed.
import "C"

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"image"

	"github.com/whomever000/poker-common"
	"github.com/whomever000/poker-common/window"
	"github.com/whomever000/poker-vision"
)

const (
	windowWidth  = 640.0
	windowHeight = 441.0
)

var img image.Image

func init() {

	// Set custom file loader.
	// This loads files from static data which is compiled into the application.
	pokervision.SetFileLoader(&fileLoader{})

	// Load reference file.
	if err := LoadReferences(); err != nil {
		panic("Failed to load references")
	}
}

// Attach attaches to a window by the specified name.
func Attach(windowName string) error {

	// Find and attach to window.
	win, err := window.Attach(windowName)
	if err != nil {
		log.Printf("error: Failed to attach to window '%v'. %v", windowName, err)
		return err
	}

	// Get window name.
	name, err := win.Name()
	if err != nil {
		log.Printf("warning: Failed to get window name. %v", err)
	}
	strs := strings.Split(name, " - ")
	if len(strs) < 3 {
		log.Printf("warning: Window name has an unexpected format. %v. %v", name, err)
	} else {
		name = strs[0]
	}

	// Get process name.
	process, err := win.Process()
	if err != nil {
		log.Printf("warning: Failed to get process name. %v", err)
	}

	// Resize window.
	err = win.Resize(windowWidth, windowHeight)
	if err != nil {
		log.Printf("error: Failed to resize window %v", err)
	}

	log.Printf("Attached to window '%v' of process '%v'", name, process)
	return nil
}

// NewHand waits for a new hand to start, then returns the initial hand JSON
// structure.
func NewHand() string {

	fmt.Println("Waiting for new round")
	waitForNewHand()
	fmt.Println("New round")

	h := new(poker.Hand)

	h.Client = client()
	h.Table = table()
	h.HandID = handID()
	h.Date = date()
	h.Button = button()
	h.SmallBlind = smallBlind()
	h.BigBlind = bigBlind()
	h.Players = players()

	b, err := json.MarshalIndent(h, "", "	")
	if err != nil {
		log.Printf("error: Failed to encode JSON. %v", err)
		return ""
	}

	fmt.Println(string(b))

	return string(b)
}

func main() {

	Attach("Halley")

	NewHand()

	/*
		r, _ := os.Open("./img.png")
		defer r.Close()
		img, _ := png.Decode(r)

		for i := 1; i <= 6; i++ {
			fmt.Println(PlayerName(img, poker.PlayerPosition(i)))
		}
	*/
}

// client returns the name of the client.
func client() string {
	return "PokerStars"
}

// table returns the details of the table.
func table() poker.Table {

	var table poker.Table

	// Get window name.
	name, err := window.Get().Name()
	if err != nil {
		log.Printf("warning: Failed to get window name. %v", err)
	}

	strs := strings.Split(name, " - ")
	if len(strs) < 3 {
		log.Printf("error: Expected three or more substrings in window name. "+
			"name=%v", name)
		return table
	}

	table.Name = strs[0]
	table.Stakes, err = poker.ParseStakes(strs[1])
	if err != nil {
		log.Printf("error: Failed to parse table stakes. %v", err)
	}
	// TODO: allow other table sizes.
	table.Size = 6
	table.Game, err = poker.ParseGame(strs[2])
	if err != nil {
		log.Printf("error: Failed to parse game. %v", err)
	}

	return table
}

// handID returns current hand ID.
func handID() int {
	return 42
}

// date returns the time the hand was started.
func date() poker.Date {
	return poker.Date(time.Now())
}

// button returns the current button position.
func button() poker.PlayerPosition {
	return ButtonPosition(img)
}

// smallBlind returns the small blind position.
func smallBlind() poker.PlayerPosition {

	btn := button()
	active := ActivePlayers(img)

	for i := 0; i < 6; i++ {
		pos := poker.NextPlayerPosition(btn, 6)
		for a := 0; a < len(active); a++ {
			if active[i] == int(pos)-1 {
				return pos
			}
		}
	}

	return 0
}

// bigBlind returns the big blind position.
func bigBlind() poker.PlayerPosition {

	sb := smallBlind()
	active := ActivePlayers(img)

	for i := 0; i < 6; i++ {
		pos := poker.NextPlayerPosition(sb, 6)
		for a := 0; a < len(active); a++ {
			if active[i] == int(pos)-1 {
				return pos
			}
		}
	}

	return 0
}

func players() []poker.Player {

	sync := make(chan bool, 6)

	players := make([]poker.Player, 6)
	for i := 0; i < 6; i++ {
		index := i
		go func() {
			name, _ := PlayerName(img, poker.PlayerPosition(index+1))
			stack, _ := Stack(img, poker.PlayerPosition(index+1))

			players[index] = poker.Player{Name: name, Stack: stack}
			sync <- true
		}()
	}

	for i := 0; i < 6; i++ {
		<-sync
	}

	return players
}

// waitForNewHand waits for a new hand.
func waitForNewHand() {

	numActive := 0
	lowestNum := 6

	for {
		// Grab image from window.
		getImage()

		// Has number of active players decreased?
		numActive = len(ActivePlayers(img))
		if numActive < lowestNum {
			lowestNum = numActive

			// Has number of active players increased?
		} else if numActive > lowestNum {
			// This is a new hand.
			// Wait for 'Post BB' to disappear.
			time.Sleep(time.Millisecond * 1000)
			getImage()
			return
		}

		time.Sleep(time.Millisecond * 100)
	}
}

func getImage() {
	var err error
	img, err = window.Get().Image()
	if err != nil {
		panic("Could not get image from window")
	}
}
