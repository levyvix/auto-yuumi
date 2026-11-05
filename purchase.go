package main

import (
	"fmt"
	"image"
	"log"
	"time"
)

func inputAllowed() bool {
	return focused() && !desktopStop.Load() && (!desktopMode || !desktopPaused.Load())
}
func tapShopKey() error {
	if !inputAllowed() {
		return fmt.Errorf("compra cancelada por foco ou pausa")
	}
	if err := emitKey(0x2D, false); err != nil {
		return err
	} // X
	time.Sleep(80 * time.Millisecond)
	return emitKey(0x2D, true)
}
func shopVisible(frame image.Image, c Config) bool {
	if c.ShopMarker == nil || c.ShopMarker.img == nil {
		return false
	}
	_, ok := match(frame, c.ShopMarker, c.Threshold)
	return ok
}

// One attempt, never repeat purchases automatically while a shop stays open.
func buyRecommended(c Config, bounds image.Rectangle, capture func() (*image.RGBA, error), open bool) error {
	if !inputAllowed() {
		return fmt.Errorf("compra cancelada por foco ou pausa")
	}
	frame, err := capture()
	if err != nil {
		return err
	}
	if !shopVisible(frame, c) {
		if !open {
			return fmt.Errorf("aba Recomendados nao confirmada; nenhum clique enviado")
		}
		if err := tapShopKey(); err != nil {
			return err
		}
		time.Sleep(400 * time.Millisecond)
		frame, err = capture()
		if err != nil {
			return err
		}
	}
	if !shopVisible(frame, c) {
		return fmt.Errorf("loja nao reconhecida apos X; nenhum clique enviado")
	}
	if !inputAllowed() {
		return fmt.Errorf("compra cancelada por foco ou pausa")
	}
	target := image.Pt(c.ShopTarget[0], c.ShopTarget[1])
	if !target.In(frame.Bounds()) {
		return fmt.Errorf("alvo de compra fora da tela")
	}
	if err := clickBase(target.Add(bounds.Min)); err != nil {
		return err
	}
	log.Print("Botao direito enviado ao recomendado da direita; compra depende de ouro, espaco e acesso a loja")
	time.Sleep(250 * time.Millisecond)
	if inputAllowed() {
		frame, err = capture()
		if err != nil {
			return err
		}
		if shopVisible(frame, c) {
			return tapShopKey()
		}
	}
	return nil
}
