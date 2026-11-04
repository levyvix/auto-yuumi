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
	if a, _ := b.step(9*time.Second, true, false); a != "" {
		t.Fatal("recall antecipado")
	}
	if a, _ := b.step(time.Second, true, false); a != "recall" {
		t.Fatal(a)
	}
	if a, blocked := b.step(8*time.Second, true, false); a != "" || !blocked {
		t.Fatal("W durante recall")
	}
	if a, _ := b.step(time.Second, true, false); a != "wait" {
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
func TestNoReturnWhileAttached(t *testing.T) {
	var b baseReturn
	for i := 0; i < 10; i++ {
		if a, blocked := b.step(time.Second, false, true); a != "" || blocked {
			t.Fatal(a)
		}
	}
}
