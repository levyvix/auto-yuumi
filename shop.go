package main

import "image"

// Shop markers occupy opposite corners of the minimap. Detect the green
// marker rather than assuming a team side from the champion portraits.
func alliedShop(frame image.Image) (image.Point, bool) {
	if frame.Bounds().Dx() != 1920 || frame.Bounds().Dy() != 1080 {
		return image.Point{}, false
	}
	candidates := []image.Rectangle{image.Rect(1879, 769, 1902, 789), image.Rect(1602, 1050, 1625, 1074)}
	scores := [2]int{}
	for i, r := range candidates {
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				rr, gg, bb, _ := frame.At(x, y).RGBA()
				red, g, b := int(rr>>8), int(gg>>8), int(bb>>8)
				if g > 65 && g > red+6 && g > b+25 {
					scores[i]++
				}
			}
		}
	}
	best, other := 0, 1
	if scores[1] > scores[0] {
		best, other = 1, 0
	}
	if scores[best] < 6 || scores[best] < scores[other]*2+3 {
		return image.Point{}, false
	}
	r := candidates[best]
	return image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2), true
}
