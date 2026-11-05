package main

import (
	"strings"
	"testing"
)

func TestOwnLifeRejectsMissingData(t *testing.T) {
	for _, tc := range []struct {
		json        string
		known, dead bool
	}{
		{`{}`, false, false},
		{`{"activePlayer":{"championStats":{"currentHealth":0,"maxHealth":700}},"allPlayers":[]}`, false, false},
		{`{"activePlayer":{"championStats":{"currentHealth":695,"maxHealth":700}},"allPlayers":[{}]}`, true, false},
		{`{"activePlayer":{"championStats":{"currentHealth":0,"maxHealth":700}},"allPlayers":[{}]}`, true, true},
	} {
		state, err := parseOwnLife(strings.NewReader(tc.json))
		if err != nil || state.Known != tc.known || state.Dead != tc.dead {
			t.Fatalf("%s: %+v %v", tc.json, state, err)
		}
	}
}

func TestPurchaseOnDeathAndNewBaseVisit(t *testing.T) {
	var p purchaseEvents
	alive := ownLife{Known: true}
	dead := ownLife{Known: true, Dead: true}
	for i := 0; i < 2; i++ {
		if reason := p.next(alive, true); reason != "" {
			t.Fatal("base sem confirmacao")
		}
	}
	if reason := p.next(alive, true); reason != "chegada a base" {
		t.Fatal(reason)
	}
	for i := 0; i < 20; i++ {
		if reason := p.next(alive, true); reason != "" {
			t.Fatal("compra repetida na mesma visita")
		}
	}
	if reason := p.next(dead, true); reason != "morte" {
		t.Fatal(reason)
	}
	for i := 0; i < 20; i++ {
		if reason := p.next(dead, false); reason != "" {
			t.Fatal("compra repetida durante morte")
		}
	}
	for i := 0; i < 2; i++ {
		p.next(alive, true)
	}
	if reason := p.next(alive, true); reason != "chegada a base" {
		t.Fatal("renascimento na base ignorado")
	}
	for i := 0; i < 3; i++ {
		p.next(alive, false)
	}
	for i := 0; i < 2; i++ {
		p.next(alive, true)
	}
	if reason := p.next(alive, true); reason != "chegada a base" {
		t.Fatal("nova visita ignorada")
	}
	if reason := p.next(ownLife{}, true); reason != "" {
		t.Fatal("compra durante carregamento ou API indisponivel")
	}
}
