package main

import "image"

var basePatches = []image.Rectangle{image.Rect(1627, 1008, 1670, 1050), image.Rect(1852, 788, 1893, 828)}

// Compare the permanent structures in both bases, not the green shop marker.
func alliedShop(frame image.Image) (image.Point, bool) {
	if frame.Bounds().Dx() != 1920 || frame.Bounds().Dy() != 1080 {
		return image.Point{}, false
	}
	blue, red := [2]int{}, [2]int{}
	for i, r := range basePatches {
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				rr, gg, bb, _ := frame.At(x, y).RGBA()
				r, g, b := int(rr>>8), int(gg>>8), int(bb>>8)
				if b > 75 && g > 65 && b > r+30 && g > r+20 {
					blue[i]++
				}
				if r > 80 && r > g+35 && r > b+25 {
					red[i]++
				}
			}
		}
	}
	southwest := blue[0] - red[0]
	northeast := blue[1] - red[1]
	if blue[0] >= 15 && red[1] >= 15 && southwest-northeast >= 35 {
		return image.Pt(1615, 1043), true
	}
	if blue[1] >= 15 && red[0] >= 15 && northeast-southwest >= 35 {
		return image.Pt(1890, 779), true
	}
	return image.Point{}, false
}

type baseLocator struct {
	window            uintptr
	locked, candidate image.Point
	votes             int
}

func (b *baseLocator) observe(window uintptr, frame image.Image) (image.Point, bool) {
	if window != b.window {
		*b = baseLocator{window: window}
	}
	if b.votes >= 5 {
		return b.locked, true
	}
	point, ok := alliedShop(frame)
	if !ok {
		b.votes = 0
		return image.Point{}, false
	}
	if point != b.candidate {
		b.candidate = point
		b.votes = 0
	}
	b.votes++
	if b.votes >= 5 {
		b.locked = point
		return point, true
	}
	return image.Point{}, false
}
