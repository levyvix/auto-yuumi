package main

import (
	"image"
	"image/draw"
	"image/png"
	"os"
	"testing"
)

func TestUpgradePriorityAndUnavailableRanks(t *testing.T) {
	f, err := os.Open("assets/level_up.png")
	if err != nil {
		t.Fatal(err)
	}
	asset, err := png.Decode(f)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	marker := &Entry{img: asset}
	for mask := 0; mask < 16; mask++ {
		frame := image.NewRGBA(image.Rect(0, 0, 1920, 1080))
		want := ""
		for i, ability := range upgradePriority {
			if mask&(1<<i) == 0 {
				continue
			}
			if want == "" {
				want = ability.name
			}
			draw.Draw(frame, image.Rect(ability.x, 965, ability.x+19, 980), asset, asset.Bounds().Min, draw.Src)
		}
		got, ok := nextUpgrade(frame, marker, 0.92)
		if ok != (want != "") || (ok && got.name != want) {
			t.Fatalf("set %04b: got %s/%v, want %s", mask, got.name, ok, want)
		}
	}
}

func TestUpgradeArrowScreenCalibration(t *testing.T) {
	f, err := os.Open("captures/loja-aberta.jpg")
	if os.IsNotExist(err) {
		t.Skip("referencia local ausente")
	}
	if err != nil {
		t.Fatal(err)
	}
	frame, _, err := image.Decode(f)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	f, err = os.Open("assets/level_up.png")
	if err != nil {
		t.Fatal(err)
	}
	asset, err := png.Decode(f)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	ability, ok := nextUpgrade(frame, &Entry{img: asset}, 0.92)
	if !ok || ability.name != "Q" {
		t.Fatalf("nivel 1: %s %v", ability.name, ok)
	}
}
