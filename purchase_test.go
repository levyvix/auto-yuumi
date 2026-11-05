package main

import (
	"image"
	"image/png"
	"os"
	"testing"
)

func TestShopRecognizedOnlyWhenRecommendedHeaderVisible(t *testing.T) {
	f, err := os.Open("assets/shop_open.png")
	if err != nil {
		t.Fatal(err)
	}
	marker, err := png.Decode(f)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	config := Config{Threshold: 0.92, ShopMarker: &Entry{img: marker, Region: []int{460, 99, 130, 15}}}
	for _, tc := range []struct {
		path string
		want bool
	}{{"captures/loja-aberta.jpg", true}, {"captures/yuumi-agarrada.png", false}} {
		f, err := os.Open(tc.path)
		if os.IsNotExist(err) {
			t.Skip("referencias locais ausentes")
		}
		if err != nil {
			t.Fatal(err)
		}
		frame, _, err := image.Decode(f)
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		if got := shopVisible(frame, config); got != tc.want {
			t.Fatalf("%s: loja=%v", tc.path, got)
		}
	}
	if shopVisible(image.NewRGBA(image.Rect(0, 0, 1920, 1080)), config) {
		t.Fatal("loja reconhecida em tela vazia")
	}
}
