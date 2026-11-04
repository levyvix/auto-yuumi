package main

import (
	"encoding/json"
	"log"
	"os"
	"runtime"
	"strings"
	"sync/atomic"
	"syscall"
	"unsafe"
)

var desktopMode bool
var desktopStop atomic.Bool
var desktopPaused atomic.Bool
var desktopE atomic.Bool
var desktopECheckbox uintptr
var desktopLabel, desktopButton uintptr
var createWindow = user32.NewProc("CreateWindowExW")
var setWindowText = user32.NewProc("SetWindowTextW")
var destroyWindow = user32.NewProc("DestroyWindow")

func wide(s string) *uint16          { p, _ := syscall.UTF16PtrFromString(s); return p }
func setText(hwnd uintptr, s string) { setWindowText.Call(hwnd, uintptr(unsafe.Pointer(wide(s)))) }

type windowClass struct {
	Size, Style                        uint32
	Procedure                          uintptr
	ClassExtra, WindowExtra            int32
	Instance, Icon, Cursor, Background uintptr
	Menu, Name                         *uint16
	SmallIcon                          uintptr
}
type windowMessage struct {
	Window         uintptr
	Message        uint32
	WParam, LParam uintptr
	Time           uint32
	X, Y           int32
	Private        uint32
}

func nativeChild(class, text string, style uint32, x, y, w, h int, parent, id uintptr) uintptr {
	child, _, _ := createWindow.Call(0, uintptr(unsafe.Pointer(wide(class))), uintptr(unsafe.Pointer(wide(text))), uintptr(style|0x50000000), uintptr(x), uintptr(y), uintptr(w), uintptr(h), parent, id, 0, 0)
	font, _, _ := syscall.NewLazyDLL("gdi32.dll").NewProc("GetStockObject").Call(17)
	user32.NewProc("SendMessageW").Call(child, 0x30, font, 1)
	return child
}

func desktopProcedure(hwnd uintptr, message uint32, wparam, lparam uintptr) uintptr {
	switch message {
	case 0x111: // WM_COMMAND
		switch wparam & 0xffff {
		case 1:
			desktopPaused.Store(!desktopPaused.Load())
		case 2:
			destroyWindow.Call(hwnd)
		case 3:
			checked, _, _ := user32.NewProc("SendMessageW").Call(desktopECheckbox, 0xF0, 0, 0)
			desktopE.Store(checked == 1)
			data, err := os.ReadFile("config.json")
			if err == nil {
				var config map[string]any
				if json.Unmarshal(data, &config) == nil {
					config["auto_e"] = checked == 1
					if updated, err := json.MarshalIndent(config, "", "  "); err == nil {
						if err := os.WriteFile("config.json", updated, 0644); err != nil {
							log.Print("Nao foi possivel salvar E: ", err)
						}
					}
				}
			}
			log.Printf("E automatico a cada 25s: %v", checked == 1)
		}
		return 0
	case 0x113: // WM_TIMER: UI updates stay on the window thread.
		if desktopStop.Load() {
			setText(desktopLabel, "Controle encerrado")
			setText(desktopButton, "Encerrado")
		} else if desktopPaused.Load() {
			setText(desktopLabel, "Pausado")
			setText(desktopButton, "Retomar")
		} else if !focused() {
			setText(desktopLabel, "Aguardando o jogo")
			setText(desktopButton, "Pausar")
		} else {
			setText(desktopLabel, "Ativo")
			setText(desktopButton, "Pausar")
		}
		return 0
	case 0x10: // WM_CLOSE
		desktopStop.Store(true)
		destroyWindow.Call(hwnd)
		return 0
	case 2: // WM_DESTROY
		desktopStop.Store(true)
		user32.NewProc("PostQuitMessage").Call(0)
		return 0
	}
	value, _, _ := user32.NewProc("DefWindowProcW").Call(hwnd, uintptr(message), wparam, lparam)
	return value
}

func runDesktopApp() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	desktopMode = true
	if data, err := os.ReadFile("config.json"); err == nil {
		var config Config
		if json.Unmarshal(data, &config) == nil {
			desktopE.Store(config.AutoE)
		}
	}
	instance, _, _ := syscall.NewLazyDLL("kernel32.dll").NewProc("GetModuleHandleW").Call(0)
	cursor, _, _ := user32.NewProc("LoadCursorW").Call(0, 32512)
	icon, _, _ := user32.NewProc("LoadIconW").Call(0, 32512)
	wc := windowClass{Procedure: syscall.NewCallback(desktopProcedure), Instance: instance, Cursor: cursor, Icon: icon, Background: 16, Name: wide("AutoYuumiDesktop")}
	wc.Size = uint32(unsafe.Sizeof(wc))
	atom, _, err := user32.NewProc("RegisterClassExW").Call(uintptr(unsafe.Pointer(&wc)))
	if atom == 0 {
		log.Print("Criar aplicativo: ", err)
		return
	}
	hwnd, _, err := createWindow.Call(0, uintptr(unsafe.Pointer(wc.Name)), uintptr(unsafe.Pointer(wide("Auto Yuumi "+strings.TrimSpace(appVersion)))), 0x00CA0000, 0x80000000, 0x80000000, 420, 220, 0, 0, instance, 0)
	if hwnd == 0 {
		log.Print("Abrir aplicativo: ", err)
		return
	}
	desktopLabel = nativeChild("STATIC", "Aguardando o jogo", 0, 20, 18, 365, 24, hwnd, 0)
	desktopECheckbox = nativeChild("BUTTON", "E automatico a cada 25 segundos", 0x10003, 20, 52, 365, 26, hwnd, 3)
	if desktopE.Load() {
		user32.NewProc("SendMessageW").Call(desktopECheckbox, 0xF1, 1, 0)
	}
	nativeChild("STATIC", "F8 para pausar ou retomar", 0, 20, 86, 365, 22, hwnd, 0)
	desktopButton = nativeChild("BUTTON", "Pausar", 0x10000, 20, 120, 172, 32, hwnd, 1)
	nativeChild("BUTTON", "Fechar", 0x10000, 204, 120, 172, 32, hwnd, 2)
	user32.NewProc("SetTimer").Call(hwnd, 1, 200, 0)
	user32.NewProc("ShowWindow").Call(hwnd, 5)
	user32.NewProc("ShowWindow").Call(hwnd, 5)
	user32.NewProc("UpdateWindow").Call(hwnd)
	os.Args = []string{os.Args[0], "run", "-live", "-autostart"}
	watchGameLaunch()
	go func() {
		if err := execute(); err != nil {
			log.Printf("Nao foi possivel iniciar: %v", err)
		}
		desktopStop.Store(true)
	}()
	var msg windowMessage
	for {
		result, _, _ := user32.NewProc("GetMessageW").Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if result == 0 || int32(result) == -1 {
			break
		}
		user32.NewProc("TranslateMessage").Call(uintptr(unsafe.Pointer(&msg)))
		user32.NewProc("DispatchMessageW").Call(uintptr(unsafe.Pointer(&msg)))
	}
}
