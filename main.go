package main

// // No C code needed.
import "C"

import (
	"encoding/json"
	"fmt"
	"image"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"

	"flag"

	"github.com/whomever000/poker-client-pokerstars/history"
	"github.com/whomever000/poker-client-pokerstars/vision"
	"github.com/whomever000/poker-common"
	"github.com/whomever000/poker-common/card"
	"github.com/whomever000/poker-common/window"
	"github.com/whomever000/poker-vision"
)

var (
	img    image.Image
	h      *poker.Hand
	imgSrc vision.ImageSource
)

func init() {

	log.SetLevel(log.DebugLevel)

	// Set custom file loader.
	// This loads files from static data which is compiled into the application.
	pokervision.SetFileLoader(&fileLoader{})

	// Load reference file.
	if err := vision.LoadReferences(); err != nil {
		log.Panic("failed to load references", err)
	}
}

// Attach attaches to a window by the specified name.
func Attach(windowName string) error {

	// Find and attach to window.
	win, err := window.Attach(windowName)
	if err != nil {
		log.Errorf("failed to attach to window '%v'. %v", windowName, err)
		return err
	}

	// Get window name.
	name, err := win.Name()
	if err != nil {
		log.Warnf("failed to get window name. %v", err)
	}
	strs := strings.Split(name, " - ")
	if len(strs) < 3 {
		log.Warnf("window name has an unexpected format. %v. %v", name, err)
	} else {
		name = strs[0]
	}

	// Get process name.
	process, err := win.Process()
	if err != nil {
		log.Warnf("failed to get process name. %v", err)
	}

	log.Infof("attached to window '%v' of process '%v'", name, process)
	return nil
}

////////////////////////////////////////////////////////////////////////////////
var (
	better        poker.PlayerPosition
	activePlayers []poker.PlayerPosition
	playerStacks  [6]poker.Amount
)

func main() {

	hFlag := flag.Int("h", 0, "pid of history")
	flag.Parse()
	if *hFlag != 0 {
		imgSrc = history.NewImageSource(*hFlag, true)
	} else {
		imgSrc = vision.NewDefaultImageSource()
	}

	var currPlayer poker.PlayerPosition

	// Attach to table window.
	//Attach("Halley")
	Attach("Play Money")

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
			if len(vision.ActivePlayers(img)) > 1 {

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

	log.Info("waiting for new hand")
	waitForNewHand()
	log.Info("New hand")

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

	// Wait for cards to be delt.
	time.Sleep(500 * time.Millisecond)
	getImage("waitForCardsDealt")

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
	waitImage(func() bool {
		commCards, _ = vision.CommunityCards(img)
		if numExCC == len(commCards) {
			return true
		}
		return false
	}, 500, "waitForCommCards")

	// Parse pot size.
	pot, err := vision.Pot(img)
	if err != nil {

		// TODO: comment out
		//log.Printf("error: Failed to get pot. %v", err)
		//return ""
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

	fmt.Print("Player ", pos, ": ")

	var curr poker.PlayerPosition

	waitImage(func() bool {
		curr = vision.CurrentPlayer(img)
		if curr != pos {
			return true
		}
		return false
	}, 500, "waitForAction")

	// Get player's action.
	a, err := vision.PlayerAction(img, pos)
	if err != nil {
		// TODO: uncomment
		//log.Printf("error: Failed to get player action. %v", err)
		return ""
	}

	// Get the players stack size.
	newStack, err := vision.PlayerStack(img, pos)
	if err != nil {
		fmt.Printf("error: Failed to parse player stack. %v", err)
	}
	// Calculate amount that was called/betted/raised.
	amount := playerStacks[pos-1] - newStack
	// Update player stack reference.
	playerStacks[pos-1] = newStack

	// Create action object
	switch a {
	case "actionFold":
		innerAction = poker.NewFoldAction()
		for i, p := range activePlayers {
			if p == pos {
				activePlayers = append(activePlayers[:i], activePlayers[i+1:]...)
				break
			}
		}
	case "actionCheck":
		innerAction = poker.NewCheckAction()
	case "actionCall":
		innerAction = poker.NewCallAction(amount)
	case "actionBet":
		innerAction = poker.NewBetAction(amount)
		better = pos
	case "actionRaise":
		innerAction = poker.NewRaiseAction(amount)
		better = pos

		if amount == -1 {
			for i, p := range activePlayers {
				if p == pos {
					activePlayers = append(activePlayers[:i], activePlayers[i+1:]...)
					break
				}
			}
		}
	default:
		log.Printf("error: Invalid player action: %v", a)
		// TODO: uncomment
		panic("")
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

// waitForNewHand waits for a new hand.
func waitForNewHand() {

	numActive := 0
	lowestNum := 6

	waitImage(func() bool {

		// Has number of active players decreased?
		numActive = len(vision.ActivePlayers(img))
		if numActive < lowestNum {
			lowestNum = numActive
			history.Save("newLow")

			// Has number of active players increased?
		} else if numActive > lowestNum {

			// This is a new hand.

			// Wait for table to be cleared.
			// Wait for all players to become active.
			// This does not happen at the exact same time.
			time.Sleep(time.Millisecond * 1500)
			getImage("waitForTableClear_playersActive")
			activePlayers = vision.ActivePlayers(img)
			return true
		}

		return false

	}, 500, "waitForNewHand")
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

// Get a new image
func getImage(descr string) {
	img = imgSrc.Get()
	history.Save(descr)
}

// Get a new image until condition is met
func waitImage(f func() bool, interval int, descr string) {
	if img == nil {
		getImage(descr)
	}

	for !f() {
		time.Sleep(time.Millisecond * time.Duration(interval))
		img = imgSrc.Get()
	}
	history.Save(descr)
}
