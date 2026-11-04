package main

import "time"

type baseReturn struct {
	phase                   string
	elapsed                 time.Duration
	deadFrames, aliveFrames int
}

func (b *baseReturn) step(dt time.Duration, alive, attached bool) (string, bool) {
	if b.phase == "" {
		if alive || attached {
			b.deadFrames = 0
			return "", false
		}
		b.deadFrames++
		if b.deadFrames < 3 {
			return "", true
		}
		b.phase = "walking"
		b.elapsed = 0
		return "walk", true
	}
	// Reevaluate before advancing any return timer or issuing recall.
	if attached {
		*b = baseReturn{}
		return "resume", false
	}
	if alive {
		b.aliveFrames++
		if b.aliveFrames >= 3 {
			*b = baseReturn{}
			return "resume", false
		}
		return "", true
	}
	b.aliveFrames = 0
	b.elapsed += dt
	switch b.phase {
	case "walking":
		if b.elapsed >= 10*time.Second {
			b.phase = "recalling"
			b.elapsed = 0
			return "recall", true
		}
	case "recalling":
		if b.elapsed >= 9*time.Second {
			b.phase = "waiting"
			b.elapsed = 0
			return "wait", true
		}
	}
	return "", true
}
