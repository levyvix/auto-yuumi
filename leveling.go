package main

import (
	"fmt"
	"image"
	"time"
)

type abilityUpgrade struct {
	name string
	scan uint16
	x    int
}

var upgradePriority = []abilityUpgrade{{"R", 0x13, 949}, {"Q", 0x10, 817}, {"E", 0x12, 905}, {"W", 0x11, 861}}

// The gold upgrade arrow is only shown when the game allows that rank.
func nextUpgrade(frame image.Image, marker *Entry, threshold float64) (abilityUpgrade, bool) {
	if marker == nil || marker.img == nil {
		return abilityUpgrade{}, false
	}
	for _, ability := range upgradePriority {
		candidate := *marker
		candidate.Region = []int{ability.x, 965, 19, 15}
		if _, ok := match(frame, &candidate, threshold); ok {
			return ability, true
		}
	}
	return abilityUpgrade{}, false
}

func levelAbility(ability abilityUpgrade) error {
	if !inputAllowed() {
		return fmt.Errorf("avanco de habilidade cancelado por foco ou pausa")
	}
	if down(0x11) {
		return fmt.Errorf("Ctrl ja pressionado pelo usuario")
	}
	if err := emitKey(0x1D, false); err != nil {
		return err
	} // left Ctrl
	defer emitKey(0x1D, true)
	time.Sleep(40 * time.Millisecond)
	if !inputAllowed() {
		return fmt.Errorf("avanco de habilidade cancelado por foco ou pausa")
	}
	if err := emitKey(ability.scan, false); err != nil {
		return err
	}
	time.Sleep(80 * time.Millisecond)
	if err := emitKey(ability.scan, true); err != nil {
		_ = emitKey(ability.scan, true)
		return err
	}
	return nil
}
