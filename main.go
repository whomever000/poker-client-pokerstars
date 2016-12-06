package main

// // No C code needed.
import "C"

import (
	"encoding/json"
	"fmt"
	"image"
	"log"
	"strings"
	"time"

	"github.com/whomever000/poker-common"
	"github.com/whomever000/poker-common/card"
	"github.com/whomever000/poker-common/window"
	"github.com/whomever000/poker-vision"
)

const (
	windowWidth  = 640.0
	windowHeight = 441.0
)

var img image.Image
var h *poker.Hand

var handid = 1

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
	// err = win.Resize(windowWidth, windowHeight)
	// if err != nil {
	// 	log.Printf("error: Failed to resize window %v", err)
	// }

	log.Printf("Attached to window '%v' of process '%v'", name, process)
	return nil
}

////////////////////////////////////////////////////////////////////////////////
var (
	better        poker.PlayerPosition
	activePlayers []poker.PlayerPosition
	playerStacks  [6]poker.Amount
)

func main() {

	var currPlayer poker.PlayerPosition

	// Attach to table window.
	Attach("Halley")

	// Handle hands.
	for {

		// Wait for new hand.
		// Wait for pocket cards to be delt.
		NewHand()

		// Prepare for new hand.
		currPlayer = nextActivePlayer(h.BigBlind)
		better = currPlayer

		// Handle betting rounds.
		for bettingRound := 0; bettingRound < 4; bettingRound++ {

			// Wait for new betting round.
			// Wait for community cards to be delt.
			NewBettingRound(bettingRound)

			// More than 1 player still active?
			if len(ActivePlayers(img)) > 1 {

			bettingRoundLoop:
				for {

					// Wait for player action.
					NewPlayerAction(currPlayer)

					// Consider next player.
					next := poker.NextPlayerPosition(currPlayer, 6)

					// Consider next active player.
					currPlayer = nextActivePlayer(currPlayer)

					// Check if betting round is done.
					// Next active player is the better? (i.e. end of round)
					if currPlayer == better {
						break
					}
					// A player between current and next active player is the
					// better? (i.e. end of round).
					for next != currPlayer {
						if next == better {
							break bettingRoundLoop
						}

						next = poker.NextPlayerPosition(next, 6)
					}
				}

				currPlayer = nextActivePlayer(h.Button)
				better = currPlayer

			} else {
				fmt.Println("Only 1 left")
			}

			//fmt.Println(returnHand())
		}
		fmt.Println(h)
	}
}

func returnHand() string {
	b, err := json.MarshalIndent(h, "", "	")
	if err != nil {
		log.Printf("error: Failed to encode JSON. %v", err)
		return ""
	}

	return string(b)
}

// NewHand waits for a new hand to start, then returns the initial hand JSON
// structure.
func NewHand() string {

	log.Println("Waiting for new hand")
	waitForNewHand()
	log.Println("New hand")

	// Create new hand object and populate with initial meta-data.
	h = new(poker.Hand)
	h.Client = client()
	h.Table = table()
	h.HandID = handID()
	h.Date = date()
	h.Button = button()
	h.SmallBlind = smallBlind()
	h.BigBlind = bigBlind()
	h.ThisPlayer = thisPlayer()
	h.Players = players()

	// Return JSON encoded hand.
	return returnHand()
}

// NewBettingRound waits for community cards to be delt, then returns the hand
// JSON structure with added betting round information.
func NewBettingRound(bettingRound int) string {

	log.Println("Waiting for new betting round")

	var (
		numExCC   int
		commCards []card.Card
		round     poker.Round
	)

	// Determine the expected number of community cards.
	switch bettingRound {
	case 0:
		numExCC = 0
	case 1:
		numExCC = 3
	case 2:
		numExCC = 4
	case 3:
		numExCC = 5
	case 4:
		log.Panicf("unexpected betting round %v, expected 0,1,2 or 3",
			bettingRound)
	}

	// Wait for the expected number of community cards to be delt.
	for numExCC != len(commCards) {
		commCards, _ = CommunityCards(img)
		getImage()
		time.Sleep(time.Millisecond * 100)
	}

	// Parse pot size.
	pot, err := Pot(img)
	if err != nil {
		log.Printf("error: Failed to get pot. %v", err)
		return ""
	}

	// Initialize round object.
	round.Cards = commCards
	round.Pot = pot

	// Add it to hand.
	h.Rounds = append(h.Rounds, round)

	log.Println("New betting round")

	// Return JSON encoded hand.
	return returnHand()
}

// NewPlayerAction waits for the player to perform an action, then returns the
// hand JSON structure with added action information.
func NewPlayerAction(pos poker.PlayerPosition) string {

	var (
		action      poker.PlayerAction
		innerAction poker.Action
	)

	fmt.Print("Player", pos, ": ")

	getImage()

	// Wait for player to loose his turn.
	for CurrentPlayer(img) == pos {
		getImage()
		time.Sleep(time.Millisecond * 100)
	}

	// The player's name field shows the action - read it.
	a, err := PlayerName(img, pos)
	if err != nil {
		// TODO: uncomment
		//log.Printf("error: Failed to get player action. %v", err)
		return ""
	}

	// Get the players stack size.
	newStack, err := PlayerStack(img, pos)
	if err != nil {
		fmt.Printf("error: Failed to parse player stack. %v", err)
	}
	// Calculate amount that was called/betted/raised.
	amount := playerStacks[pos-1] - newStack
	// Update player stack reference.
	playerStacks[pos-1] = newStack

	// Create action object
	// TODO: Read call/bet/raise amounts.
	a = strings.ToLower(a)
	switch a {
	case "fold":
		innerAction = poker.NewFoldAction()
		for i, p := range activePlayers {
			if p == pos {
				activePlayers = append(activePlayers[:i], activePlayers[i+1:]...)
				break
			}
		}
	case "check":
		innerAction = poker.NewCheckAction()
	case "call":
		innerAction = poker.NewCallAction(amount)
	case "bet":
		innerAction = poker.NewBetAction(amount)
		better = pos
	case "raise":
		innerAction = poker.NewRaiseAction(amount)
		better = pos
	default:
		log.Printf("error: Invalid player action: %v", a)
	}

	// Initialize PlayerAction object
	action.Position = pos
	action.Action = innerAction

	fmt.Println(innerAction, "\tStack:", playerStacks[pos-1])

	// Insert into last round.
	currRound := len(h.Rounds)
	h.Rounds[currRound-1].Actions = append(h.Rounds[currRound-1].Actions, action)

	// Return JSON encoded hand.
	return returnHand()
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
	ret := handid
	handid++
	return ret
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

	return nextActivePlayer(h.Button)
}

// bigBlind returns the big blind position.
func bigBlind() poker.PlayerPosition {

	return nextActivePlayer(h.SmallBlind)
}

func thisPlayer() *poker.PlayerCards {

	cards, err := PocketCards(img)
	if err != nil {
		log.Printf("error: Failed to get pocket cards. %v", err)
		return nil
	}

	return &poker.PlayerCards{4, cards}
}

func players() []poker.Player {

	sync := make(chan bool, 6)

	players := make([]poker.Player, 6)
	for i := 0; i < 6; i++ {
		index := i
		go func() {
			name, _ := PlayerName(img, poker.PlayerPosition(index+1))
			stack, _ := PlayerStack(img, poker.PlayerPosition(index+1))

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

// waitForNewHand waits for a new hand.
func waitForNewHand() {

	numActive := 0
	lowestNum := 6

	first := true

	for {
		// Grab image from window.
		getImage()

		if first {
			/*
				for i := 0; i < 6; i++ {
					val, _ := PlayerName(img, poker.PlayerPosition(i+1))

					fmt.Println(val)

				}
				fmt.Println("---------------")*/
		}
		first = false
		//val, _ := Pot(img)
		//val := CurrentPlayer(img)
		//fmt.Println(val)

		// Has number of active players decreased?
		numActive = len(ActivePlayers(img))
		fmt.Println(numActive)
		if numActive < lowestNum {
			lowestNum = numActive

			// Has number of active players increased?
		} else if numActive > lowestNum {

			// This is a new hand.

			// Wait for all players to become active.
			// This does not happen at the exact same time.
			time.Sleep(time.Millisecond * 500)

			// Store active players.
			getImage()
			activePlayers = ActivePlayers(img)

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

func nextActivePlayer(pos poker.PlayerPosition) poker.PlayerPosition {

	for i := 0; i < 6; i++ {
		pos = poker.NextPlayerPosition(pos, 6)

		for a := 0; a < len(activePlayers); a++ {
			if activePlayers[a] == pos {
				return pos
			}
		}
	}

	return 0
}
