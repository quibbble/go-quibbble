package server

import (
	"fmt"
	"math/rand"
	"time"
)

func generateName() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprintf("%s-%s-%d", adjectives[r.Intn(len(adjectives))], nouns[r.Intn(len(nouns))], r.Intn(99))
}

var adjectives = []string{
	"fat",
	"mad",
	"icy",
	"apt",
	"big",
	"hot",
	"dry",
	"ill",
	"lax",
	"ply",
	"sad",
	"shy",
	"sly",
	"wet",
	"wry",
	"fit",
	"fun",
	"new",
	"pro",
	"odd",
	"tan",
	"old",
	"toy",
	"red",
	"coy",
}

var nouns = []string{
	"cat",
	"hat",
	"bat",
	"dog",
	"bow",
	"cap",
	"cow",
	"egg",
	"doe",
	"fox",
	"fog",
	"gas",
	"gem",
	"jam",
	"hog",
	"car",
	"van",
	"wig",
	"sea",
	"pig",
	"fig",
	"ore",
	"inn",
	"oak",
	"owl",
	"spy",
}
