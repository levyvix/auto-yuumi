package main

import (
	"golang.org/x/sys/windows"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

var windowProcess = user32.NewProc("GetWindowThreadProcessId")

func isGameWindow(hwnd uintptr) bool {
	if hwnd == 0 {
		return false
	}
	var pid uint32
	windowProcess.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	process, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(process)
	var name [32768]uint16
	size := uint32(len(name))
	if windows.QueryFullProcessImageName(process, 0, &name[0], &size) != nil {
		return false
	}
	return strings.EqualFold(filepath.Base(windows.UTF16ToString(name[:size])), "League of Legends.exe")
}

var enumLock sync.Mutex
var enumFound uintptr
var gameWindowCallback = syscall.NewCallback(func(hwnd, param uintptr) uintptr {
	visible, _, _ := user32.NewProc("IsWindowVisible").Call(hwnd)
	if visible == 0 || !isGameWindow(hwnd) {
		return 1
	}
	var r struct{ Left, Top, Right, Bottom int32 }
	windowRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
	if r.Right-r.Left < 400 || r.Bottom-r.Top < 300 {
		return 1
	}
	enumFound = hwnd
	return 0
})

func findGameWindow() uintptr {
	enumLock.Lock()
	defer enumLock.Unlock()
	enumFound = 0
	user32.NewProc("EnumWindows").Call(gameWindowCallback, 0)
	return enumFound
}

// Activate only on a new game window, never repeatedly after the user Alt-Tabs.
func watchGameLaunch() {
	previous := findGameWindow()
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			if desktopStop.Load() {
				return
			}
			current := findGameWindow()
			if current != 0 && current != previous && !desktopPaused.Load() {
				iconic, _, _ := user32.NewProc("IsIconic").Call(current)
				if iconic != 0 {
					user32.NewProc("ShowWindowAsync").Call(current, 9)
				}
				ok, _, _ := user32.NewProc("SetForegroundWindow").Call(current)
				if ok != 0 {
					log.Print("Nova janela da partida detectada e ativada")
				} else {
					log.Print("Nova partida detectada; Windows nao autorizou trazer a janela para frente")
				}
			}
			previous = current
		}
	}()
}
