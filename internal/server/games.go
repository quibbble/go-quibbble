package server

import (
	bg "github.com/quibbble/go-boardgame"
	carcassonne "github.com/quibbble/go-carcassonne"
	connect4 "github.com/quibbble/go-connect4"
	indigo "github.com/quibbble/go-indigo"
	stratego "github.com/quibbble/go-stratego"
	tictactoe "github.com/quibbble/go-tictactoe"
	tsuro "github.com/quibbble/go-tsuro"
)

var (
	carcassonneBuilder = carcassonne.Builder{}
	connect4Builder    = connect4.Builder{}
	indigoBuilder      = indigo.Builder{}
	strategoBuilder    = stratego.Builder{}
	tictactoeBuilder   = tictactoe.Builder{}
	tsuroBuilder       = tsuro.Builder{}
)

var games = map[string]bg.BoardGameWithBGNBuilder{
	carcassonneBuilder.Key(): &carcassonneBuilder,
	connect4Builder.Key():    &connect4Builder,
	indigoBuilder.Key():      &indigoBuilder,
	strategoBuilder.Key():    &strategoBuilder,
	tictactoeBuilder.Key():   &tictactoeBuilder,
	tsuroBuilder.Key():       &tsuroBuilder,
}
