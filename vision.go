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
	"github.com/whomever000/poker-vision"
)

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

	// The string includes 'Pot:', so remove this before parsing.
	pot = strings.Replace(pot, "Pot:", "", -1)

	return poker.ParseAmount(pot)
}

// Stack returns a player's stack.
func Stack(img image.Image, position poker.PlayerPosition) (poker.Amount, error) {

	if position < 1 {
		return 0, fmt.Errorf("Invalid player: %v", int(position))
	}
	stack := m.Match(fmt.Sprintf("plStack%v", int(position)-1), img)

	// All in is represented as -1.
	if strings.Contains(stack, "All") {
		return poker.NewAmount(-1), nil
	}

	return poker.ParseAmount(stack)
}

// Name returns a player's name.
func PlayerName(img image.Image, position poker.PlayerPosition) (string, error) {

	if position < 1 {
		return "", fmt.Errorf("Invalid player: %v", int(position))
	}

	return m.Match(fmt.Sprintf("plName%v", int(position)-1), img), nil
}

func PocketCards(img image.Image) ([]card.Card, error) {
	val0 := m.Match("pocketValue0", img)
	col0 := m.Match("pocketColor0", img)

	val1 := m.Match("pocketValue1", img)
	col1 := m.Match("pocketColor1", img)

	c1, _ := card.ParseCard(fmt.Sprintf("%v%v", val0[2:], col0[:1]))
	c2, _ := card.ParseCard(fmt.Sprintf("%v%v", val1[3:], col1[:1]))

	fmt.Print(c1)
	fmt.Print(c2)

	return nil, nil
}

func ActivePlayers(img image.Image) (ret []int) {
	for i := 0; i < 6; i++ {
		active := m.Match("plActive"+strconv.Itoa(i), img)
		if len(active) != 0 {
			ret = append(ret, i)
		}
	}
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
