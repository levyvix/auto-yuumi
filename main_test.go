package main

import (
	"encoding/json"
	"image"
	"image/color"
	"os"
	"testing"
	"unsafe"
)

func TestNativeWLayout(t *testing.T) {
	i := wInput(false)
	if unsafe.Sizeof(i) != 40 || unsafe.Offsetof(i.Data) != 8 {
		t.Fatal("layout INPUT incompativel com Win64")
	}
	k := (*nativeKeyboard)(unsafe.Pointer(&i.Data[0]))
	if i.Kind != 1 || k.Scan != 0x11 || k.Flags != 8 || k.VK != 0 {
		t.Fatal("evento W invalido")
	}
	i = wInput(true)
	k = (*nativeKeyboard)(unsafe.Pointer(&i.Data[0]))
	if k.Flags != 10 {
		t.Fatal("soltura de W invalida")
	}
}

func TestPortraitReferences(t *testing.T) {
	f, err := os.Open("captures/yuumi-solta-aprendida.jpg")
	if os.IsNotExist(err) {
		t.Skip("referencia local ausente")
	}
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	frame, _, err := image.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("config.json")
	if err != nil {
		t.Fatal(err)
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatal(err)
	}
	for _, e := range c.Allies {
		if _, ok := locateAlly(frame, e, c.Threshold); !ok {
			t.Errorf("barra de vida nao reconhecida: %s", e.Name)
		}
		gray := image.NewGray(frame.Bounds())
		for y := e.Health[1]; y < e.Health[1]+e.Health[3]; y++ {
			for x := e.Health[0]; x < e.Health[0]+e.Health[2]; x++ {
				gray.Set(x, y, frame.At(x, y))
			}
		}
		if _, ok := locateAlly(gray, e, c.Threshold); ok {
			t.Errorf("retrato cinza sintetico aceito: %s", e.Name)
		}
	}
}

func TestCalibrationReferences(t *testing.T) {
	load := func(path string) image.Image {
		f, err := os.Open(path)
		if os.IsNotExist(err) {
			t.Skip("referencias locais ausentes")
		}
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		img, _, err := image.Decode(f)
		if err != nil {
			t.Fatal(err)
		}
		return img
	}
	attached := &Entry{img: load("assets/attached.png"), Region: []int{855, 997, 30, 26}}
	detached := &Entry{img: load("assets/detached.png"), Region: []int{855, 997, 30, 26}}
	for _, tc := range []struct {
		path     string
		attached bool
	}{
		{"captures/yuumi-agarrada.png", true},
		{"captures/yuumi-solta-aprendida.jpg", false},
	} {
		frame := load(tc.path)
		_, a := match(frame, attached, 0.92)
		_, d := match(frame, detached, 0.92)
		if a != tc.attached || d == tc.attached {
			t.Fatalf("%s: attached=%v detached=%v", tc.path, a, d)
		}
	}
}

func TestGate(t *testing.T) {
	for _, s := range []struct{ a, d, r, w bool }{{true, true, true, false}, {false, false, true, false}, {false, true, false, false}, {false, true, true, true}} {
		if canAttach(s.a, s.d, s.r) != s.w {
			t.Fatal(s)
		}
	}
}
func TestMatch(t *testing.T) {
	frame := image.NewRGBA(image.Rect(0, 0, 20, 20))
	asset := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			c := color.RGBA{uint8(80 + x*50), uint8(90 + y*50), 200, 255}
			asset.Set(x, y, c)
			frame.Set(10+x, 12+y, c)
		}
	}
	e := &Entry{img: asset, Region: []int{8, 10, 8, 8}}
	if p, ok := match(frame, e, 1); !ok || p != image.Pt(11, 13) {
		t.Fatalf("%v %v", p, ok)
	}
	e.Region = []int{0, 0, 5, 5}
	if _, ok := match(frame, e, 1); ok {
		t.Fatal("fora da regiao")
	}
}
