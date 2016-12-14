package main

import (
	"bytes"
	"fmt"
	"image"
	"io"
	"log"
	"strconv"
	"strings"

	"github.com/whomever000/poker-common"
	"github.com/whomever000/poker-common/card"
	"github.com/whomever000/poker-common/window"
	"github.com/whomever000/poker-vision"
)

func VisualizeSource(img image.Image, srcs []string) image.Image {
	return m.VisualizeSource(img, srcs)
}

var m pokervision.Matcher

type fileLoader struct{}

func (l *fileLoader) Load(fileName string) io.Reader {

	if strings.HasPrefix(fileName, "./") {
		fileName = fileName[2:]
	}

	data, err := Asset(fileName)
	if err != nil {
		log.Printf("error: Failed to load file %v", fileName)
		return nil
	}

	return bytes.NewReader(data)
}

// LoadReferences loads the reference file.
func LoadReferences() error {

	var err error
	m, err = pokervision.NewMatcher("./references/refs.json")

	return err
}

// Pot returns the current pot.
func Pot(img image.Image) (poker.Amount, error) {
	pot := m.Match("pot", img)
	window.DebugImage(VisualizeSource(img, []string{"pot"}), "vision")

	// The string includes 'Pot:', so remove this before parsing.
	pot = strings.ToLower(pot)
	pot = strings.Replace(pot, "pot:", "", -1)
	pot = strings.Replace(pot, " ", "", -1)
	pot = strings.Replace(pot, "L", "1.", -1)
	pot = strings.Replace(pot, "S", "5.", -1)

	if len(pot) == 0 {
		return 0, fmt.Errorf("Failed to get pot, got empty string")
	}

	// Remote '$' as it may be misinterpreted by OCR.
	pot = pot[1:]

	return poker.ParseAmount(pot)
}

// PlayerStack returns a player's stack.
func PlayerStack(img image.Image, position poker.PlayerPosition) (poker.Amount, error) {

	if position < 1 {
		return 0, fmt.Errorf("Invalid player: %v", int(position))
	}
	p := fmt.Sprintf("plStack%v", int(position)-1)
	stack := m.Match(p, img)
	stack = strings.Replace(stack, "L", "1.", -1)
	window.DebugImage(VisualizeSource(img, []string{p}), "vision")

	// All in is represented as -1.
	if stack == "allin" {
		return poker.Amount(-1), nil
	}

	return poker.ParseAmount(stack)
}

// PlayerName returns a player's name.
func PlayerName(img image.Image, position poker.PlayerPosition) (string, error) {

	if position < 1 {
		return "", fmt.Errorf("Invalid player: %v", int(position))
	}

	p := fmt.Sprintf("plName%v", int(position)-1)
	name := m.Match(p, img)
	window.DebugImage(VisualizeSource(img, []string{p}), "vision")

	return name, nil
}

// PlayerName returns a player's name.
func PlayerAction(img image.Image, position poker.PlayerPosition) (string, error) {

	if position < 1 {
		return "", fmt.Errorf("Invalid player: %v", int(position))
	}

	p := fmt.Sprintf("plAction%v", int(position)-1)
	action := m.Match(p, img)
	window.DebugImage(VisualizeSource(img, []string{p}), "vision")

	return action, nil
}

// ActivePlayers returns active players.
func ActivePlayers(img image.Image) (ret []poker.PlayerPosition) {

	var srcs []string

	for i := 0; i < 6; i++ {

		p := fmt.Sprintf("plActive%v", i)
		srcs = append(srcs, p)

		active := m.Match(p, img)
		if len(active) != 0 {
			ret = append(ret, poker.PlayerPosition(i+1))
		}

	}

	window.DebugImage(VisualizeSource(img, srcs), "vision")

	return
}

func ButtonPosition(img image.Image) poker.PlayerPosition {
	for i := 0; i < 6; i++ {
		btn := m.Match("button"+strconv.Itoa(i), img)
		if len(btn) != 0 {
			return poker.PlayerPosition(i + 1)
		}
	}
	log.Printf("error: Unable to get button position")
	return 0
}

func PocketCards(img image.Image) ([]card.Card, error) {
	val0 := m.Match("pocketValue0", img)
	col0 := m.Match("pocketColor0", img)

	val1 := m.Match("pocketValue1", img)
	col1 := m.Match("pocketColor1", img)

	if len(val0) == 0 || len(col0) == 0 {
		val0 = "someInvalidCard"
		col0 = "someInvalidCard"
	}
	if len(val1) == 0 || len(col1) == 0 {
		val1 = "someInvalidCard"
		col1 = "someInvalidCard"
	}

	c1, err1 := card.ParseCard(fmt.Sprintf("%v%v", val0[3:], col0[:1]))
	c2, err2 := card.ParseCard(fmt.Sprintf("%v%v", val1[3:], col1[:1]))

	if err1 != nil {
		return []card.Card{c1, c2}, err1
	}

	return []card.Card{c1, c2}, err2
}

func CommunityCards(img image.Image) ([]card.Card, error) {

	var cards []card.Card

	for i := 0; i < 5; i++ {

		val := m.Match(fmt.Sprintf("commValue%v", i), img)
		col := m.Match(fmt.Sprintf("commColor%v", i), img)

		if len(val) == 0 || len(col) == 0 {
			break
		}

		c, err := card.ParseCard(fmt.Sprintf("%v%v", val[3:], col[:1]))
		if err != nil {
			return nil, err
		}

		cards = append(cards, c)
	}

	num := len(cards)
	if num != 0 && num != 3 && num != 4 && num != 5 {
		return nil, fmt.Errorf("error: Unexpected amount of community cards %v", num)
	}

	return cards, nil
}

func CurrentPlayer(img image.Image) poker.PlayerPosition {
	for i := 0; i < 6; i++ {
		active := m.Match("plCurrent"+strconv.Itoa(i), img)
		if active != "" {
			return poker.PlayerPosition(i + 1)
		}
	}

	return 0
}
