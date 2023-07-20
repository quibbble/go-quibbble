package server

import (
	bg "github.com/quibbble/go-boardgame"
	tictactoe "github.com/quibbble/go-boardgame/examples/tictactoe"
	carcassonne "github.com/quibbble/go-carcassonne"
	codenames "github.com/quibbble/go-codenames"
	connect4 "github.com/quibbble/go-connect4"
	stratego "github.com/quibbble/go-stratego"
	tsuro "github.com/quibbble/go-tsuro"
)

var (
	carcassonneBuilder = carcassonne.Builder{}
	codenamesBuilder   = codenames.Builder{}
	connect4Builder    = connect4.Builder{}
	strategoBuilder    = stratego.Builder{}
	tictactoeBuilder   = tictactoe.Builder{}
	tsuroBuilder       = tsuro.Builder{}
)

var games = map[string]bg.BoardGameWithBGNBuilder{
	carcassonneBuilder.Key(): &carcassonneBuilder,
	codenamesBuilder.Key():   &codenamesBuilder,
	connect4Builder.Key():    &connect4Builder,
	strategoBuilder.Key():    &strategoBuilder,
	tictactoeBuilder.Key():   &tictactoeBuilder,
	tsuroBuilder.Key():       &tsuroBuilder,
}
