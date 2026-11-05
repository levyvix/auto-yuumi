package main

import (
	"image"
	"testing"
	"time"
)

func TestFountainWalkStopsAndRestartsTimer(t *testing.T) {
	var walk fountainWalk
	start := time.Unix(100, 0)
	base := image.Pt(1615, 1043)
	if _, ok := walk.target(start, true, false, false, true, base); ok {
		t.Fatal("movimento imediato")
	}
	first, ok := walk.target(start.Add(20*time.Second), true, false, false, true, base)
	if !ok || first != image.Pt(1620, 1053) {
		t.Fatalf("primeiro movimento: %v %v", first, ok)
	}
	second, ok := walk.target(start.Add(40*time.Second), true, false, false, true, base)
	if !ok || second == first {
		t.Fatalf("pontos nao alternaram: %v %v", second, ok)
	}
	if _, ok := walk.target(start.Add(time.Minute), true, true, false, true, base); ok {
		t.Fatal("movimento com ADC vivo")
	}
	if _, ok := walk.target(start.Add(2*time.Minute), true, false, false, true, base); ok {
		t.Fatal("nova espera deve reiniciar o intervalo")
	}
	if _, ok := walk.target(start.Add(3*time.Minute), true, false, false, false, base); ok {
		t.Fatal("movimento sem sinal de base")
	}
}
