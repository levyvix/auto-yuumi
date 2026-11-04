package main

import (
	"testing"
	"time"
)

func TestADCReturn(t *testing.T) {
	var b baseReturn
	for i := 0; i < 2; i++ {
		if action, _ := b.step(0, false, false); action != "" {
			t.Fatal("morte sem confirmacao")
		}
	}
	if a, blocked := b.step(0, false, false); a != "walk" || !blocked {
		t.Fatal(a, blocked)
	}
	if a, _ := b.step(9*time.Second, false, false); a != "" {
		t.Fatal("recall antecipado")
	}
	if a, _ := b.step(time.Second, false, false); a != "recall" {
		t.Fatal(a)
	}
	if a, blocked := b.step(8*time.Second, false, false); a != "" || !blocked {
		t.Fatal("W durante recall")
	}
	if a, _ := b.step(time.Second, false, false); a != "wait" {
		t.Fatal(a)
	}
	for i := 0; i < 2; i++ {
		if _, blocked := b.step(0, true, false); !blocked {
			t.Fatal("renascimento sem confirmacao")
		}
	}
	if a, blocked := b.step(0, true, false); a != "resume" || blocked {
		t.Fatal(a, blocked)
	}
}

func TestRespawnCancelsEveryReturnPhase(t *testing.T) {
	for _, phase := range []string{"walking", "recalling", "waiting"} {
		t.Run(phase, func(t *testing.T) {
			b := baseReturn{phase: phase, elapsed: 10 * time.Second}
			for i := 0; i < 2; i++ {
				if a, blocked := b.step(time.Second, true, false); a != "" || !blocked {
					t.Fatal("acao emitida durante confirmacao do ADC vivo", a)
				}
			}
			if a, blocked := b.step(time.Second, true, false); a != "resume" || blocked || b.phase != "" {
				t.Fatal("retorno nao cancelado", a, blocked, b)
			}
		})
	}
}

func TestRespawnFlickerAndManualAttachment(t *testing.T) {
	b := baseReturn{phase: "walking"}
	b.step(0, true, false)
	b.step(0, true, false)
	b.step(0, false, false)
	if a, _ := b.step(0, true, false); a != "" {
		t.Fatal("leitura intermitente cancelou retorno")
	}
	if a, blocked := b.step(0, false, true); a != "resume" || blocked {
		t.Fatal("agarrada manualmente nao cancelou retorno")
	}
}
func TestNoReturnWhileAttached(t *testing.T) {
	var b baseReturn
	for i := 0; i < 10; i++ {
		if a, blocked := b.step(time.Second, false, true); a != "" || blocked {
			t.Fatal(a)
		}
	}
}
