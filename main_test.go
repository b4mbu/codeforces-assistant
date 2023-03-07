package main

import (
	"testing"
)

// func TestLoadContest(t *testing.T) {
// 	res, err := loadContest("1794")
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	fmt.Println(res)
// }

// func TestGetProblemSamples(t *testing.T) {
// 	res, err := getProblemSamples("1794", "D")
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	fmt.Println(res)
// }

func TestLoadContest(t *testing.T) {
	err := loadContest("1794")
	if err != nil {
		t.Error(err)
	}
}
