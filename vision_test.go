package main

import (
	"testing"

	"image/png"
	"os"

	"github.com/whomever000/poker/hand"
)

const refFile = "./references/refs.json"

// setup sets up for testing.
func setup(t *testing.T) {
	err := LoadReferences(refFile)
	if err != nil {
		t.Fatalf("Failed to load refFile: %v", err)
	}
}

// Pot /////////////////////////////////////////////////////////////////////////

type testPairPot struct {
	image string
	pot   hand.Amount
}

var testsPot = []testPairPot{
	{"./testdata/testImg1.png", hand.NewAmount(1.06)},
	{"./testdata/testImg2.png", hand.NewAmount(3.23)},
}

func TestPot(t *testing.T) {
	setup(t)

	for i := 0; i < len(testsPot); i++ {

		// Open image file.
		f, err := os.Open(testsPot[i].image)
		if err != nil {
			t.Errorf("Failed to open image file: %v", err)
			continue
		}

		// Decode PNG.
		img, err := png.Decode(f)
		f.Close()
		if err != nil {
			t.Errorf("Failed to decode image file: %v", err)
			continue
		}

		amount, err := Pot(img)
		if err != nil {
			t.Errorf("Failed to parse: %v", err)
			continue
		}

		if amount != testsPot[i].pot {
			t.Errorf("For %v expected %v, got %v", testsPot[i].image,
				testsPot[i].pot, amount)
		}
	}
}

// Stack ///////////////////////////////////////////////////////////////////////

type testPairStack struct {
	image  string
	stacks []hand.Amount
}

var testsStack = []testPairStack{
	{"./testdata/testImg1.png",
		[]hand.Amount{
			hand.NewAmount(4),
			hand.NewAmount(4.48),
			hand.NewAmount(1.94),
			hand.NewAmount(1.21),
			hand.NewAmount(3.26),
			hand.NewAmount(1.51),
		},
	},
	{"./testdata/testImg2.png", []hand.Amount{
		hand.NewAmount(4.79),
		hand.NewAmount(5.05),
		hand.NewAmount(3.99),
		hand.NewAmount(4.94),
		hand.NewAmount(-1),
		hand.NewAmount(5.90),
	},
	},
}

func TestStack(t *testing.T) {
	setup(t)

	for i := 0; i < len(testsStack); i++ {

		// Open image file.
		f, err := os.Open(testsStack[i].image)
		if err != nil {
			t.Errorf("Failed to open image file: %v", err)
			continue
		}

		// Decode PNG.
		img, err := png.Decode(f)
		f.Close()
		if err != nil {
			t.Errorf("Failed to decode image file: %v", err)
			continue
		}

		// TODO: Should not be limited to 6 seats.
		for p := 0; p < 6; p++ {
			amount, err := Stack(img, hand.PlayerPosition(p+1))
			if err != nil {
				t.Errorf("Failed to parse: %v", err)
				continue
			}

			if amount != testsStack[i].stacks[p] {
				t.Errorf("For %v expected %v, got %v", testsStack[i].image,
					testsStack[i].stacks[p], amount)
			}
		}
	}
}

// Name ////////////////////////////////////////////////////////////////////////

type testPairName struct {
	image string
	names []string
}

var testsName = []testPairName{
	{"./testdata/testImg1.png",
		[]string{
			"LuisTirelli",
			"fixeer19",
			"deniscabar",
			"oolrunnings",
			"803555",
			"skendroshen",
		},
	},
	{"./testdata/testImg2.png",
		[]string{
			"FhePiedPokeI",
			"Sheryman",
			"frymek",
			"belyann",
			"bananadlr",
			"AlexCBus",
		},
	},
}

func TestName(t *testing.T) {
	setup(t)

	for i := 0; i < len(testsName); i++ {

		// Open image file.
		f, err := os.Open(testsStack[i].image)
		if err != nil {
			t.Errorf("Failed to open image file: %v", err)
			continue
		}

		// Decode PNG.
		img, err := png.Decode(f)
		f.Close()
		if err != nil {
			t.Errorf("Failed to decode image file: %v", err)
			continue
		}

		// TODO: Should not be limited to 6 seats.
		for p := 0; p < 6; p++ {
			name, err := Name(img, hand.PlayerPosition(p+1))
			if err != nil {
				t.Errorf("Failed to parse: %v", err)
				continue
			}

			if name != testsName[i].names[p] {
				t.Errorf("For %v expected %v, got %v", testsStack[i].image,
					testsName[i].names[p], name)
			}
		}
	}
}
