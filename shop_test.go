package main

import (
	"image"
	"image/color"
	"os"
	"testing"
)

func TestShopBothSidesAndAmbiguity(t *testing.T) {
	for _, p := range []image.Point{image.Pt(1889, 779), image.Pt(1613, 1062)} {
		frame := image.NewRGBA(image.Rect(0, 0, 1920, 1080))
		for y := p.Y - 3; y <= p.Y+3; y++ {
			for x := p.X - 3; x <= p.X+3; x++ {
				frame.Set(x, y, color.RGBA{80, 180, 30, 255})
			}
		}
		found, ok := alliedShop(frame)
		if !ok || abs(found.X-p.X) > 2 || abs(found.Y-p.Y) > 2 {
			t.Fatalf("%v: %v %v", p, found, ok)
		}
	}
	if _, ok := alliedShop(image.NewRGBA(image.Rect(0, 0, 1920, 1080))); ok {
		t.Fatal("loja inventada")
	}
}
func TestShopReference(t *testing.T) {
	f, err := os.Open("captures/yuumi-agarrada.png")
	if os.IsNotExist(err) {
		t.Skip("referencia ausente")
	}
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	frame, _, err := image.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	p, ok := alliedShop(frame)
	if !ok || p.X < 1800 {
		t.Fatalf("loja da referencia: %v %v", p, ok)
	}
}
