//go:build windows

package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/kbinani/screenshot"
	"image"
	_ "image/jpeg"
	"image/png"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

//go:embed version.txt
var appVersion string

type Entry struct {
	Target    *[2]int `json:"target,omitempty"`
	Health    []int   `json:"health_region,omitempty"`
	Name      string  `json:"name"`
	Enabled   *bool   `json:"enabled"`
	Template  string  `json:"template"`
	Region    []int   `json:"region"`
	Offset    [2]int  `json:"click_offset"`
	Threshold float64 `json:"threshold"`
	MinChroma float64 `json:"min_chroma"`
	img       image.Image
}

func locateAlly(frame image.Image, e *Entry, threshold float64) (image.Point, bool) {
	if e.Target == nil {
		return match(frame, e, threshold)
	}
	if len(e.Health) != 4 {
		return image.Point{}, false
	}
	r := image.Rect(e.Health[0], e.Health[1], e.Health[0]+e.Health[2], e.Health[1]+e.Health[3])
	if r.Empty() || !r.In(frame.Bounds()) || !image.Pt(e.Target[0], e.Target[1]).In(frame.Bounds()) {
		return image.Point{}, false
	}
	green := 0
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			rr, gg, bb, _ := frame.At(x, y).RGBA()
			red, g, b := int(rr>>8), int(gg>>8), int(bb>>8)
			if g > 65 && g > red+15 && g > b+10 {
				green++
			}
		}
	}
	return image.Pt(e.Target[0], e.Target[1]), green >= 3
}

type Config struct {
	AutoE     bool              `json:"auto_e"`
	Monitor   int               `json:"monitor"`
	Poll      float64           `json:"poll_seconds"`
	Retry     float64           `json:"retry_seconds"`
	Frames    int               `json:"confirm_frames"`
	Threshold float64           `json:"threshold"`
	Signals   map[string]*Entry `json:"signals"`
	Allies    []*Entry          `json:"allies"`
}

var user32 = syscall.NewLazyDLL("user32.dll")
var asyncKey = user32.NewProc("GetAsyncKeyState")
var setCursor = user32.NewProc("SetCursorPos")
var getCursor = user32.NewProc("GetCursorPos")
var sendInput = user32.NewProc("SendInput")
var foreground = user32.NewProc("GetForegroundWindow")
var windowText = user32.NewProc("GetWindowTextW")
var windowRect = user32.NewProc("GetWindowRect")

// Win64 INPUT contains an aligned 32-byte union; KEYBDINPUT occupies its start.
type nativeInput struct {
	Kind    uint32
	Padding uint32
	Data    [32]byte
}
type nativeKeyboard struct {
	VK    uint16
	Scan  uint16
	Flags uint32
	Time  uint32
	Extra uintptr
}

func wInput(release bool) nativeInput {
	i := nativeInput{Kind: 1}
	k := (*nativeKeyboard)(unsafe.Pointer(&i.Data[0]))
	k.Scan = 0x11    // physical W scan code
	k.Flags = 0x0008 // KEYEVENTF_SCANCODE
	if release {
		k.Flags |= 0x0002
	}
	return i
}
func emitW(release bool) error {
	return emitKey(0x11, release)
}
func emitKey(scan uint16, release bool) error {
	i := wInput(release)
	(*nativeKeyboard)(unsafe.Pointer(&i.Data[0])).Scan = scan
	n, _, err := sendInput.Call(1, uintptr(unsafe.Pointer(&i)), unsafe.Sizeof(i))
	if n != 1 {
		return fmt.Errorf("Windows nao aceitou tecla scan=0x%X (release=%v): %v; verifique se jogo e app usam o mesmo nivel de permissao", scan, release, err)
	}
	return nil
}
func castW(point image.Point) error {
	if !focused() || desktopStop.Load() || (desktopMode && desktopPaused.Load()) {
		return fmt.Errorf("jogo sem foco ou controle pausado")
	}
	var old struct{ X, Y int32 }
	if ok, _, err := getCursor.Call(uintptr(unsafe.Pointer(&old))); ok == 0 {
		return fmt.Errorf("ler cursor: %v", err)
	}
	if ok, _, err := setCursor.Call(uintptr(point.X), uintptr(point.Y)); ok == 0 {
		return fmt.Errorf("mover cursor: %v", err)
	}
	defer func() {
		if focused() {
			setCursor.Call(uintptr(old.X), uintptr(old.Y))
		}
	}()
	time.Sleep(150 * time.Millisecond)
	if !focused() || down(0x77) || desktopStop.Load() || (desktopMode && desktopPaused.Load()) {
		return fmt.Errorf("tentativa cancelada por foco ou pausa")
	}
	var current struct{ X, Y int32 }
	getCursor.Call(uintptr(unsafe.Pointer(&current)))
	if abs(int(current.X)-point.X) > 6 || abs(int(current.Y)-point.Y) > 6 {
		return fmt.Errorf("cursor saiu do alvo antes de W")
	}
	if err := emitW(false); err != nil {
		return err
	}
	time.Sleep(80 * time.Millisecond)
	if err := emitW(true); err != nil {
		_ = emitW(true)
		return err
	}
	time.Sleep(200 * time.Millisecond)
	return nil
}
func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func clickBase(point image.Point) error {
	if !focused() || desktopStop.Load() || (desktopMode && desktopPaused.Load()) {
		return fmt.Errorf("retorno cancelado por foco ou pausa")
	}
	var old struct{ X, Y int32 }
	getCursor.Call(uintptr(unsafe.Pointer(&old)))
	setCursor.Call(uintptr(point.X), uintptr(point.Y))
	defer func() {
		if focused() {
			setCursor.Call(uintptr(old.X), uintptr(old.Y))
		}
	}()
	time.Sleep(150 * time.Millisecond)
	if !focused() || desktopStop.Load() || (desktopMode && desktopPaused.Load()) {
		return fmt.Errorf("retorno cancelado por foco ou pausa")
	}
	for _, flags := range []uint32{0x0008, 0x0010} {
		i := nativeInput{}
		*(*uint32)(unsafe.Pointer(&i.Data[12])) = flags
		if n, _, err := sendInput.Call(1, uintptr(unsafe.Pointer(&i)), unsafe.Sizeof(i)); n != 1 {
			return fmt.Errorf("clique de retorno rejeitado: %v", err)
		}
		time.Sleep(80 * time.Millisecond)
	}
	return nil
}

func foregroundMonitor() (int, error) {
	hwnd, _, _ := foreground.Call()
	var r struct{ Left, Top, Right, Bottom int32 }
	ok, _, _ := windowRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
	if ok == 0 {
		return 0, fmt.Errorf("nao foi possivel identificar a janela ativa; use -monitor N")
	}
	window := image.Rect(int(r.Left), int(r.Top), int(r.Right), int(r.Bottom))
	best, area := 0, 0
	for i := 0; i < screenshot.NumActiveDisplays(); i++ {
		overlap := window.Intersect(screenshot.GetDisplayBounds(i))
		if size := overlap.Dx() * overlap.Dy(); !overlap.Empty() && size > area {
			best, area = i+1, size
		}
	}
	if best == 0 {
		return 0, fmt.Errorf("janela ativa fora dos monitores; use -monitor N")
	}
	return best, nil
}

func down(k uintptr) bool { v, _, _ := asyncKey.Call(k); return v&0x8000 != 0 }
func focused() bool {
	hwnd, _, _ := foreground.Call()
	return isGameWindow(hwnd)
}

// RGB mean absolute error. Threshold is a tolerance, not a probability.
func match(frame image.Image, e *Entry, threshold float64) (image.Point, bool) {
	if e.Threshold > 0 {
		threshold = e.Threshold
	}
	region := image.Rect(e.Region[0], e.Region[1], e.Region[0]+e.Region[2], e.Region[1]+e.Region[3]).Intersect(frame.Bounds())
	w, h := e.img.Bounds().Dx(), e.img.Bounds().Dy()
	budget := (1 - threshold) * 255 * float64(w*h*3)
	for y := region.Min.Y; y+h <= region.Max.Y; y++ {
		for x := region.Min.X; x+w <= region.Max.X; x++ {
			sum := 0.0
			reject := false
			for ty := 0; ty < h && !reject; ty++ {
				for tx := 0; tx < w; tx++ {
					r, g, b, _ := frame.At(x+tx, y+ty).RGBA()
					tr, tg, tb, _ := e.img.At(e.img.Bounds().Min.X+tx, e.img.Bounds().Min.Y+ty).RGBA()
					for _, d := range [3]int{int(r>>8) - int(tr>>8), int(g>>8) - int(tg>>8), int(b>>8) - int(tb>>8)} {
						if d < 0 {
							d = -d
						}
						sum += float64(d)
					}
					if sum > budget {
						reject = true
						break
					}
				}
			}
			if !reject {
				if e.MinChroma > 0 {
					chroma := 0.0
					for cy := 0; cy < h; cy++ {
						for cx := 0; cx < w; cx++ {
							r, g, b, _ := frame.At(x+cx, y+cy).RGBA()
							hi, lo := max(r, g, b), min(r, g, b)
							chroma += float64((hi - lo) >> 8)
						}
					}
					if chroma/float64(w*h) < e.MinChroma {
						continue
					}
				}
				return image.Pt(x+w/2, y+h/2), true
			}
		}
	}
	return image.Point{}, false
}
func canAttach(a, d, r bool) bool { return !a && d && r }
func save(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
func execute() error {
	command := "run"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	live := fs.Bool("live", false, "Enviar W")
	autostart := fs.Bool("autostart", false, "Iniciar simulacao sem F8")
	seconds := fs.Int("seconds", 0, "Encerrar teste apos N segundos (0 sem limite)")
	name := fs.String("name", "", "Nome do asset")
	rect := fs.String("rect", "", "x,y,largura,altura")
	source := fs.String("source", "", "Recortar PNG existente, sem capturar a tela")
	monitor := fs.Int("monitor", -1, "Monitor: 0 automatico; 1, 2... manual")
	if len(os.Args) > 1 {
		if err := fs.Parse(os.Args[2:]); err != nil {
			return err
		}
	}
	if command == "monitors" {
		for i := 0; i < screenshot.NumActiveDisplays(); i++ {
			b := screenshot.GetDisplayBounds(i)
			fmt.Printf("Monitor %d: %dx%d, posicao (%d,%d)\n", i+1, b.Dx(), b.Dy(), b.Min.X, b.Min.Y)
		}
		return nil
	}
	data, err := os.ReadFile("config.json")
	if err != nil {
		return err
	}
	if *autostart && *live && !desktopMode {
		return fmt.Errorf("autostart permitido somente sem live")
	}
	var c Config
	if err = json.Unmarshal(data, &c); err != nil {
		return err
	}
	if len(c.Allies) > 1 {
		c.Allies = c.Allies[:1]
	} // ADC only.
	if *monitor >= 0 {
		c.Monitor = *monitor
	}
	if command == "capture" && *source != "" {
		if *name == "" || strings.ContainsAny(*name, "/\\.:") {
			return fmt.Errorf("informe -name valido")
		}
		f, err := os.Open(*source)
		if err != nil {
			return err
		}
		img, _, err := image.Decode(f)
		f.Close()
		if err != nil {
			return err
		}
		var x, y, w, h int
		if n, _ := fmt.Sscanf(*rect, "%d,%d,%d,%d", &x, &y, &w, &h); n != 4 || w <= 0 || h <= 0 {
			return fmt.Errorf("use -rect x,y,largura,altura")
		}
		r := image.Rect(x, y, x+w, y+h)
		if !r.In(img.Bounds()) {
			return fmt.Errorf("recorte fora da imagem")
		}
		cropped := image.NewRGBA(image.Rect(0, 0, w, h))
		for dy := 0; dy < h; dy++ {
			for dx := 0; dx < w; dx++ {
				cropped.Set(dx, dy, img.At(x+dx, y+dy))
			}
		}
		return save(filepath.Join("assets", *name+".png"), cropped)
	}
	if command == "capture" {
		fmt.Println("Captura em 5 segundos. Deixe o jogo em primeiro plano.")
		time.Sleep(5 * time.Second)
	}
	if c.Monitor == 0 {
		if command == "run" || command == "status" {
			fmt.Println("Aguardando o jogo em primeiro plano para escolher o monitor.")
			for !focused() {
				if desktopStop.Load() {
					return nil
				}
				time.Sleep(100 * time.Millisecond)
			}
		}
		c.Monitor, err = foregroundMonitor()
		if err != nil {
			return err
		}
	}
	if c.Monitor < 1 || c.Monitor > screenshot.NumActiveDisplays() {
		return fmt.Errorf("monitor invalido")
	}
	bounds := screenshot.GetDisplayBounds(c.Monitor - 1)
	fmt.Printf("Monitor %d: %dx%d, posicao (%d,%d)\n", c.Monitor, bounds.Dx(), bounds.Dy(), bounds.Min.X, bounds.Min.Y)
	capture := func() (*image.RGBA, error) { return screenshot.CaptureRect(bounds) }
	if command == "capture" {
		frame, err := capture()
		if err != nil {
			return err
		}
		if err = save("captures/latest.png", frame); err != nil {
			return err
		}
		archive := filepath.Join("captures", "capture-"+time.Now().Format("20060102-150405.000")+".png")
		if err = save(archive, frame); err != nil {
			return err
		}
		fmt.Printf("Copia preservada: %s\n", archive)
		if *name == "" {
			fmt.Println("Salvo captures/latest.png")
			return nil
		}
		if strings.ContainsAny(*name, "/\\.:") {
			return fmt.Errorf("nome invalido")
		}
		var x, y, w, h int
		if n, _ := fmt.Sscanf(*rect, "%d,%d,%d,%d", &x, &y, &w, &h); n != 4 || w <= 0 || h <= 0 {
			return fmt.Errorf("use -rect x,y,largura,altura")
		}
		crop := image.Rect(x, y, x+w, y+h)
		if !crop.In(frame.Bounds()) {
			return fmt.Errorf("recorte fora da tela")
		}
		return save(filepath.Join("assets", *name+".png"), frame.SubImage(crop))
	}
	if command != "run" && command != "status" {
		return fmt.Errorf("use capture, status ou run")
	}
	if c.Poll <= 0 || c.Retry <= 0 || c.Frames < 1 || c.Threshold <= 0 || c.Threshold > 1 {
		return fmt.Errorf("configuracao invalida")
	}
	for _, k := range []string{"attached", "detached", "w_ready"} {
		if c.Signals[k] == nil {
			return fmt.Errorf("sinal ausente: %s", k)
		}
	}
	entries := append([]*Entry{}, c.Allies...)
	if command == "status" {
		entries = nil
	}
	for _, e := range c.Signals {
		entries = append(entries, e)
	}
	for _, e := range entries {
		if e == nil {
			return fmt.Errorf("entrada nula")
		}
		if e.Enabled != nil && !*e.Enabled {
			continue
		}
		if e.Target != nil {
			if len(e.Health) != 4 {
				return fmt.Errorf("health_region ausente: %s", e.Name)
			}
			continue
		}
		f, err := os.Open(e.Template)
		if err != nil {
			return fmt.Errorf("asset necessario: %s", e.Template)
		}
		e.img, err = png.Decode(f)
		f.Close()
		if err != nil {
			return err
		}
		if len(e.Region) != 4 {
			return fmt.Errorf("configure region [x,y,largura,altura] em %s", e.Template)
		}
		r := image.Rect(e.Region[0], e.Region[1], e.Region[0]+e.Region[2], e.Region[1]+e.Region[3])
		if r.Empty() || !r.In(image.Rect(0, 0, bounds.Dx(), bounds.Dy())) || r.Dx() < e.img.Bounds().Dx() || r.Dy() < e.img.Bounds().Dy() {
			return fmt.Errorf("region invalida: %s", e.Template)
		}
	}
	log.Printf("Simulacao=%v. F8 inicia/pausa. Pausado=%v.", !*live, !*autostart)
	paused, wasF8 := !*autostart, false
	previous, frames := -1, 0
	var last time.Time
	lastState := ""
	lastObservation := ""
	started := time.Now()
	lastE := time.Now()
	var returning baseReturn
	var baseSide baseLocator
	lastActive := time.Now()
	for {
		gameFocus := focused()
		if desktopStop.Load() {
			return nil
		}
		if desktopMode {
			paused = desktopPaused.Load()
		}
		if *seconds > 0 && time.Since(started) >= time.Duration(*seconds)*time.Second {
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
		// Loading screens and a dead Yuumi must never trigger ADC-death actions.
		self := &Entry{Target: &[2]int{800, 1050}, Health: []int{780, 1045, 270, 12}}
		if _, ready := locateAlly(frame, self, c.Threshold); !ready {
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
				if *live {
					if err := clickBase(image.Pt(bounds.Min.X+shop.X, bounds.Min.Y+shop.Y)); err != nil {
						log.Print(err)
						returning = baseReturn{}
					}
				}
			case "recall":
				log.Print("Retorno: enviando B; W e E bloqueados durante canalizacao")
				if *live && focused() && !desktopStop.Load() && (!desktopMode || !desktopPaused.Load()) {
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
				if *live && !attachedNow && focused() && !desktopStop.Load() && (!desktopMode || !desktopPaused.Load()) {
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
		} else if *live && time.Since(lastE) >= 25*time.Second {
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
		if command == "run" && !*live {
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
			if *live {
				log.Printf("TENTATIVA REAL de W em %s (%v)", e.Name, point)
			} else {
				log.Printf("SIMULACAO: usaria W em %s (%v); nenhuma tecla enviada", e.Name, point)
			}
			if *live && focused() && !down(0x77) {
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
func main() {
	if exe, err := os.Executable(); err == nil {
		if err := os.Chdir(filepath.Dir(exe)); err != nil {
			log.Print(err)
			return
		}
	}
	if err := os.MkdirAll("logs", 0755); err != nil {
		log.Print(err)
		return
	}
	logFile, err := os.OpenFile(filepath.Join("logs", "session-"+time.Now().Format("20060102-150405.000")+".log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Print(err)
		return
	}
	defer logFile.Close()
	log.SetOutput(io.MultiWriter(logFile, os.Stderr))
	log.Print("Sessao iniciada. Resultados salvos em ", logFile.Name())
	if len(os.Args) == 1 || (len(os.Args) == 2 && os.Args[1] == "--paused") {
		if len(os.Args) == 2 {
			desktopPaused.Store(true)
		}
		runDesktopApp()
		return
	}
	if err := execute(); err != nil {
		log.Fatal(err)
	}
}
