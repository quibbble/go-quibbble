package server

import (
	bg "github.com/quibbble/go-boardgame"
	carcassonne "github.com/quibbble/go-carcassonne"
	codenames "github.com/quibbble/go-codenames"
	connect4 "github.com/quibbble/go-connect4"
	tsuro "github.com/quibbble/go-tsuro"
)

var (
	carcassonneBuilder = carcassonne.Builder{}
	codenamesBuilder   = codenames.Builder{}
	connect4Builder    = connect4.Builder{}
	tsuroBuilder       = tsuro.Builder{}
)

var games = map[string]bg.BoardGameWithBGNBuilder{
	carcassonneBuilder.Key(): &carcassonneBuilder,
	codenamesBuilder.Key():   &codenamesBuilder,
	connect4Builder.Key():    &connect4Builder,
	tsuroBuilder.Key():       &tsuroBuilder,
}
