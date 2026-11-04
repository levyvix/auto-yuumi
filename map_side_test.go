package main

import (
	"image"
	"image/draw"
	"image/png"
	"os"
	"testing"
)

func mapReference(t *testing.T, side string) *image.RGBA {
	t.Helper()
	frame := image.NewRGBA(image.Rect(0, 0, 1920, 1080))
	for i, suffix := range []string{"sw", "ne"} {
		f, err := os.Open("testdata/base_" + side + "_" + suffix + ".png")
		if err != nil {
			t.Fatal(err)
		}
		crop, err := png.Decode(f)
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		draw.Draw(frame, basePatches[i], crop, crop.Bounds().Min, draw.Src)
	}
	return frame
}

func TestBaseBothRealSides(t *testing.T) {
	for _, tc := range []struct {
		side string
		want image.Point
	}{{"blue", image.Pt(1615, 1043)}, {"red", image.Pt(1890, 779)}} {
		p, ok := alliedShop(mapReference(t, tc.side))
		if !ok || p != tc.want {
			t.Fatalf("%s: %v %v", tc.side, p, ok)
		}
	}
	if _, ok := alliedShop(image.NewRGBA(image.Rect(0, 0, 1920, 1080))); ok {
		t.Fatal("base inventada em tela vazia")
	}
}

func TestBaseLocksUntilNewGame(t *testing.T) {
	blue, red := mapReference(t, "blue"), mapReference(t, "red")
	var locator baseLocator
	for i := 0; i < 4; i++ {
		if _, ok := locator.observe(100, blue); ok {
			t.Fatal("lado aceito sem cinco confirmacoes")
		}
	}
	p, ok := locator.observe(100, blue)
	if !ok || p != image.Pt(1615, 1043) {
		t.Fatal("lado azul nao confirmado")
	}
	for i := 0; i < 10; i++ {
		p, ok = locator.observe(100, red)
		if !ok || p != image.Pt(1615, 1043) {
			t.Fatal("lado mudou durante a mesma partida")
		}
	}
	if _, ok := locator.observe(200, red); ok {
		t.Fatal("nova partida herdou lado anterior")
	}
	for i := 0; i < 4; i++ {
		p, ok = locator.observe(200, red)
	}
	if !ok || p != image.Pt(1890, 779) {
		t.Fatal("novo lado nao reconhecido")
	}
}

func TestAmbiguousFramesResetConfirmation(t *testing.T) {
	var locator baseLocator
	blue := mapReference(t, "blue")
	for i := 0; i < 4; i++ {
		locator.observe(100, blue)
	}
	locator.observe(100, image.NewRGBA(image.Rect(0, 0, 1920, 1080)))
	if _, ok := locator.observe(100, blue); ok {
		t.Fatal("quadros ambiguos nao reiniciaram confirmacao")
	}
}
