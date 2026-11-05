package main

import (
	"image"
	"time"
)

const fountainInterval = 20 * time.Second

type fountainWalk struct {
	next   time.Time
	second bool
}

func (w *fountainWalk) target(now time.Time, waiting, adcAlive, attached, homeguard bool, base image.Point) (image.Point, bool) {
	var points [2]image.Point
	switch base {
	case image.Pt(1615, 1043):
		points = [2]image.Point{image.Pt(1620, 1053), image.Pt(1626, 1053)}
	case image.Pt(1890, 779):
		points = [2]image.Point{image.Pt(1887, 783), image.Pt(1893, 783)}
	}
	if !waiting || adcAlive || attached || !homeguard || points[0] == (image.Point{}) {
		*w = fountainWalk{}
		return image.Point{}, false
	}
	if w.next.IsZero() {
		w.next = now.Add(fountainInterval)
	}
	if now.Before(w.next) {
		return image.Point{}, false
	}
	p := points[0]
	if w.second {
		p = points[1]
	}
	w.second = !w.second
	w.next = now.Add(fountainInterval)
	return p, true
}
