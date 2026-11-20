package main

import (
	"golang.org/x/sys/windows"
	"log"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

var windowProcess = user32.NewProc("GetWindowThreadProcessId")
var attachThreadInput = user32.NewProc("AttachThreadInput")
var getCurrentThreadID = syscall.NewLazyDLL("kernel32.dll").NewProc("GetCurrentThreadId")

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

func activateGameWindow(hwnd uintptr) bool {
	if hwnd == 0 {
		return false
	}
	showWindowAsync := user32.NewProc("ShowWindowAsync")
	bringWindowToTop := user32.NewProc("BringWindowToTop")
	setForegroundWindow := user32.NewProc("SetForegroundWindow")
	setActiveWindow := user32.NewProc("SetActiveWindow")
	isIconic := user32.NewProc("IsIconic")
	getForegroundWindow := user32.NewProc("GetForegroundWindow")
	getThreadID, _, _ := getCurrentThreadID.Call()
	var targetPID uint32
	targetThread, _, _ := windowProcess.Call(hwnd, uintptr(unsafe.Pointer(&targetPID)))

	for attempt := 0; attempt < 8; attempt++ {
		if iconic, _, _ := isIconic.Call(hwnd); iconic != 0 {
			showWindowAsync.Call(hwnd, 9) // SW_RESTORE
		}
		foreground, _, _ := getForegroundWindow.Call()
		var foregroundPID uint32
		foregroundThread, _, _ := windowProcess.Call(foreground, uintptr(unsafe.Pointer(&foregroundPID)))
		attached := make([]uintptr, 0, 2)
		for _, thread := range []uintptr{foregroundThread, targetThread} {
			if thread != 0 && thread != getThreadID && (len(attached) == 0 || attached[0] != thread) {
				if ok, _, _ := attachThreadInput.Call(getThreadID, thread, 1); ok != 0 {
					attached = append(attached, thread)
				}
			}
		}
		bringWindowToTop.Call(hwnd)
		setForegroundWindow.Call(hwnd)
		setActiveWindow.Call(hwnd)
		for _, thread := range attached {
			attachThreadInput.Call(getThreadID, thread, 0)
		}
		if active, _, _ := getForegroundWindow.Call(); active == hwnd {
			return true
		}
		time.Sleep(200 * time.Millisecond)
	}
	return false
}

// Activate an existing or newly created game window, retrying briefly if
// Windows initially rejects the foreground request. Stop once focus succeeds.
func watchGameLaunch() {
	previous := findGameWindow()
	pending := previous
	deadline := time.Time{}
	if pending != 0 {
		deadline = time.Now().Add(2 * time.Second)
	}
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			if desktopStop.Load() {
				return
			}
			current := findGameWindow()
			if current != previous {
				previous = current
				pending = 0
				if current != 0 && !desktopPaused.Load() {
					pending = current
					deadline = time.Now().Add(2 * time.Second)
				}
			}
			if current == 0 {
				pending = 0
			}
			if pending != 0 && !desktopPaused.Load() {
				if foregroundWindow, _, _ := user32.NewProc("GetForegroundWindow").Call(); foregroundWindow == pending {
					pending = 0
				} else if time.Now().After(deadline) {
					log.Print("Windows nao autorizou focar a partida apos varias tentativas")
					pending = 0
				} else if activateGameWindow(pending) {
					log.Print("Nova janela da partida detectada e ativada")
					pending = 0
				}
			}
		}
	}()
}
