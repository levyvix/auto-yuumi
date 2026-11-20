//go:build windows

package main

import (
	"fmt"
	"image"
	"log"
	"time"
)

type automationRuntime struct {
	command   string
	live      bool
	autostart bool
	seconds   int
	config    Config
	bounds    image.Rectangle
	capture   func() (*image.RGBA, error)
}

func (runtime automationRuntime) run() error {
	c := runtime.config
	bounds := runtime.bounds
	capture := runtime.capture
	command := runtime.command
	live := runtime.live
	autostart := runtime.autostart
	seconds := runtime.seconds
	log.Printf("Simulacao=%v. F8 inicia/pausa. Pausado=%v.", !live, !autostart)
	paused, wasF8 := !autostart, false
	previous, frames := -1, 0
	var last time.Time
	lastState := ""
	lastObservation := ""
	started := time.Now()
	lastE := time.Now()
	var returning baseReturn
	var baseSide baseLocator
	shopHandled := false
	var ownState lifeCache
	var buyEvents purchaseEvents
	var lastLevel time.Time
	var fountain fountainWalk
	lastActive := time.Now()
	for {
		gameFocus := focused()
		if desktopStop.Load() {
			return nil
		}
		if desktopMode {
			paused = desktopPaused.Load()
		}
		if seconds > 0 && time.Since(started) >= time.Duration(seconds)*time.Second {
			log.Print("Teste concluido por limite de tempo")
			return nil
		}
		pressed := gameFocus && down(0x77)
		if pressed && !wasF8 {
			paused = !paused
			if desktopMode {
				desktopPaused.Store(paused)
			}
			log.Printf("Pausado=%v", paused)
		}
		wasF8 = pressed
		if paused || !gameFocus {
			fountain = fountainWalk{}
			lastActive = time.Now()
			lastE = time.Now()
			previous, frames = -1, 0
			time.Sleep(50 * time.Millisecond)
			continue
		}
		frame, err := capture()
		if err != nil {
			return err
		}
		homeguard := false
		if c.BaseMarker != nil && c.BaseMarker.img != nil {
			_, homeguard = match(frame, c.BaseMarker, c.Threshold)
		}
		if command == "run" && c.AutoBuy {
			life := ownState.read(time.Now())
			if reason := buyEvents.next(life, homeguard); reason != "" {
				log.Print("Compra automatica por ", reason)
				shopHandled = true
				if live {
					if err := buyRecommended(c, bounds, capture, true); err != nil {
						log.Print("Compra: ", err)
					}
				} else {
					log.Print("SIMULACAO: abriria X, compraria recomendado e fecharia X")
				}
				lastE = time.Now()
				lastActive = time.Now()
				previous, frames = -1, 0
				continue
			}
		}
		if command == "run" && c.AutoBuy && shopVisible(frame, c) {
			if !shopHandled {
				shopHandled = true
				if live {
					if err := buyRecommended(c, bounds, capture, false); err != nil {
						log.Print("Compra: ", err)
					}
				} else {
					log.Print("SIMULACAO: compraria o recomendado da direita")
				}
			}
			lastE = time.Now()
			lastActive = time.Now()
			previous, frames = -1, 0
			time.Sleep(time.Duration(c.Poll * float64(time.Second)))
			continue
		}
		shopHandled = false
		if command == "run" && c.AutoLevel && time.Since(lastLevel) >= 1500*time.Millisecond && !shopVisible(frame, c) {
			if ability, ok := nextUpgrade(frame, c.LevelMarker, c.Threshold); ok {
				lastLevel = time.Now()
				if live {
					var err error
					if ability.name == "R" {
						// Clicking the detected upgrade arrow avoids relying on the
						// player's Ctrl+R binding for the ultimate.
						if point, found := upgradePosition(frame, c.LevelMarker, ability, c.Threshold); found {
							err = clickLevelArrow(point.Add(bounds.Min))
						} else {
							err = fmt.Errorf("seta de R desapareceu antes do clique")
						}
					} else {
						err = levelAbility(ability)
					}
					if err != nil {
						log.Print("Habilidade: ", err)
					} else if ability.name == "R" {
						time.Sleep(250 * time.Millisecond)
						confirmation, captureErr := capture()
						if captureErr != nil {
							log.Print("Clique na seta de R enviado; nao foi possivel confirmar: ", captureErr)
						} else if _, stillAvailable := upgradePosition(confirmation, c.LevelMarker, ability, c.Threshold); stillAvailable {
							log.Print("Seta de R continua visivel depois do clique; vou tentar novamente")
						} else {
							log.Print("Ultimate evoluida; seta de R desapareceu")
						}
					} else {
						log.Printf("Ctrl+%s enviado para upar habilidade (R > Q > E > W)", ability.name)
					}
				} else {
					log.Printf("SIMULACAO: uparia %s", ability.name)
				}
				lastActive = time.Now()
				continue
			}
		}
		// Loading screens and a dead Yuumi must never trigger ADC-death actions.
		self := &Entry{Target: &[2]int{800, 1050}, Health: []int{780, 1045, 270, 12}}
		if _, ready := locateAlly(frame, self, c.Threshold); !ready {
			fountain = fountainWalk{}
			lastActive = time.Now()
			lastE = time.Now()
			previous, frames = -1, 0
			time.Sleep(time.Duration(c.Poll * float64(time.Second)))
			continue
		}
		now := time.Now()
		gameHandle, _, _ := foreground.Call()
		wasLocked := baseSide.votes >= 5 && baseSide.window == gameHandle
		shop, shopKnown := baseSide.observe(gameHandle, frame)
		if shopKnown && !wasLocked {
			log.Printf("Base aliada confirmada pelas estruturas do minimapa: %v; lado fixado nesta partida", shop)
		}
		dt := now.Sub(lastActive)
		lastActive = now
		_, attachedNow := match(frame, c.Signals["attached"], c.Threshold)
		if command == "run" && len(c.Allies) > 0 {
			_, adcAlive := locateAlly(frame, c.Allies[0], c.Threshold)
			action, blocked := returning.step(dt, adcAlive, attachedNow)
			switch action {
			case "walk":
				if !shopKnown {
					log.Print("ADC morreu: lado da base ainda nao confirmado; aguardando sem clicar")
					returning = baseReturn{}
					break
				}
				log.Printf("ADC morreu: retornando para a base aliada em %v; recall em 10s", shop)
				if live {
					if err := clickBase(image.Pt(bounds.Min.X+shop.X, bounds.Min.Y+shop.Y)); err != nil {
						log.Print(err)
						returning = baseReturn{}
					}
				}
			case "recall":
				log.Print("Retorno: enviando B; W e E bloqueados durante canalizacao")
				if live && focused() && !desktopStop.Load() && (!desktopMode || !desktopPaused.Load()) {
					if err := emitKey(0x30, false); err != nil {
						log.Print(err)
					} else {
						time.Sleep(80 * time.Millisecond)
						if err := emitKey(0x30, true); err != nil {
							log.Print(err)
						}
					}
				}
			case "wait":
				log.Print("Tempo de recall terminado: aguardando ADC vivo; recall interrompido ainda nao e detectado")
			case "resume":
				log.Print("ADC vivo ou Yuumi agarrada: retorno cancelado; retomando W somente no ADC")
				if live && !attachedNow && focused() && !desktopStop.Load() && (!desktopMode || !desktopPaused.Load()) {
					if err := emitKey(0x1F, false); err != nil {
						log.Print("Falha ao cancelar movimento/recall: ", err)
					} else {
						time.Sleep(80 * time.Millisecond)
						if err := emitKey(0x1F, true); err != nil {
							log.Print(err)
						}
					}
				}
			}
			if point, ok := fountain.target(now, c.WalkInBase && returning.phase == "waiting", adcAlive, attachedNow, homeguard && shopKnown, shop); ok {
				if live {
					if err := clickBase(point.Add(bounds.Min)); err != nil {
						log.Print("Movimento na fonte: ", err)
					} else {
						log.Printf("Movimento curto na fonte em %v; aguardando ADC", point)
					}
				} else {
					log.Printf("SIMULACAO: movimento curto na fonte em %v", point)
				}
			}
			if blocked {
				lastE = time.Now()
				previous, frames = -1, 0
				time.Sleep(time.Duration(c.Poll * float64(time.Second)))
				continue
			}
		}
		eEnabled := c.AutoE
		if desktopMode {
			eEnabled = desktopE.Load()
		}
		if !eEnabled {
			lastE = time.Now()
		} else if live && time.Since(lastE) >= 25*time.Second {
			if focused() && !desktopStop.Load() && (!desktopMode || (!desktopPaused.Load() && desktopE.Load())) {
				if err := emitKey(0x12, false); err != nil {
					log.Printf("Falha ao enviar E: %v", err)
				} else {
					time.Sleep(80 * time.Millisecond)
					if err := emitKey(0x12, true); err != nil {
						log.Printf("Falha ao soltar E: %v", err)
					} else {
						log.Print("E automatico enviado (intervalo 25s)")
					}
				}
			}
			lastE = time.Now()
		}
		_, a := match(frame, c.Signals["attached"], c.Threshold)
		_, d := match(frame, c.Signals["detached"], c.Threshold)
		_, r := match(frame, c.Signals["w_ready"], c.Threshold)
		if command == "run" && !live {
			var available []string
			for _, e := range c.Allies {
				if e.Enabled != nil && !*e.Enabled {
					continue
				}
				if _, ok := locateAlly(frame, e, c.Threshold); ok {
					available = append(available, e.Name)
				}
			}
			observation := fmt.Sprintf("agarrada=%v solta=%v W_visual=%v aliados_com_vida=%v", a, d, r, available)
			if observation != lastObservation {
				log.Print(observation)
				lastObservation = observation
			}
		}
		if command == "status" {
			state := "DESCONHECIDO (cooldown, morte, bloqueio ou imagem diferente)"
			if a && !d {
				state = "AGARRADA"
			} else if d && !a {
				state = "SOLTA"
			}
			if state != lastState {
				log.Printf("Estado visual: %s", state)
				lastState = state
			}
			time.Sleep(time.Duration(c.Poll * float64(time.Second)))
			continue
		}
		chosen := -1
		var point image.Point
		if canAttach(a, d, r) {
			for i, e := range c.Allies {
				if e.Enabled != nil && !*e.Enabled {
					continue
				}
				if p, ok := locateAlly(frame, e, c.Threshold); ok {
					chosen, point = i, p
					break
				}
			}
		}
		if chosen >= 0 && chosen == previous {
			frames++
		} else {
			frames = 0
			if chosen >= 0 {
				frames = 1
			}
		}
		previous = chosen
		if chosen >= 0 && frames >= c.Frames && time.Since(last) >= time.Duration(c.Retry*float64(time.Second)) {
			e := c.Allies[chosen]
			if live {
				log.Printf("TENTATIVA REAL de W em %s (%v)", e.Name, point)
			} else {
				log.Printf("SIMULACAO: usaria W em %s (%v); nenhuma tecla enviada", e.Name, point)
			}
			if live && focused() && !down(0x77) {
				if err := castW(image.Pt(bounds.Min.X+point.X+e.Offset[0], bounds.Min.Y+point.Y+e.Offset[1])); err != nil {
					log.Printf("FALHA NO ENVIO: %v", err)
				} else {
					log.Print("Windows aceitou W pressionado/solto; aguardando sinal visual de agarrada")
				}
			}
			last = time.Now()
			frames = 0
		}
		time.Sleep(time.Duration(c.Poll * float64(time.Second)))
	}
}
