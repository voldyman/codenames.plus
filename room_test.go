package main

import (
	"log"
	"testing"
)

func TestSelectWords(t *testing.T) {
	out := map[string]struct{}{}
	selectWords([]string{"a", "b", "c"}, 1, out)

	if len(out) != 1 {
		log.Fatal("invalid number of words selected", out)
	}
}
func TestIsSet(t *testing.T) {
	if !isSet(BoardTypeDefault, BoardTypeDefault) {
		t.Fatal("equality check failed")
	}
	if !isSet(BoardTypeDefault, BoardTypeDefault|BoardTypeCustom) {
		t.Fatal("test for a combination of two item failed")
	}

}
func TestSetsEnabled(t *testing.T) {
	defaultSets := getTotalSetsEnabled(BoardTypeDefault)
	if defaultSets != 1 {
		t.Fatal("wrong set count for one set")
	}

	compundCount := getTotalSetsEnabled(BoardTypeCustom | BoardTypeDefault)
	if compundCount != 2 {
		t.Fatal("wrong set count for two sets", compundCount)
	}
}

func TestBoardGeneration(t *testing.T) {
	tiles := generateBoard(BoardTypeDefault, TeamBlue)

	if len(tiles) != 5 {
		t.Fatal("board doesn't have 5 rows", len(tiles))
	}
	for i := 0; i < 5; i++ {
		for j := 0; j < 5; j++ {
			if tiles[i][j].Word == "" {
				t.Fatal("invalid tile")
			}
		}
	}
}
