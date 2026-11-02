//go:build darwin

// Package mainloop runs the macOS main run loop for command-line tools (no AppKit app), so that
// CoreBluetooth callbacks on the main dispatch queue are delivered. The GUI uses mygo.RunOnMain instead.
package mainloop

import (
	"runtime"
	"sync"

	"github.com/ebitengine/purego"
)

func init() { runtime.LockOSThread() } // main goroutine stays on the main thread

var (
	once         sync.Once
	mainQ        uintptr
	dispatchAsyc func(q, ctx, fn uintptr)
	cfRunLoopRun func()
	trampoline   uintptr

	mu    sync.Mutex
	next  uintptr
	tasks = map[uintptr]func(){}
)

func setup() {
	once.Do(func() {
		sys, err := purego.Dlopen("/usr/lib/libSystem.B.dylib", purego.RTLD_GLOBAL|purego.RTLD_NOW)
		if err != nil {
			panic(err)
		}
		cf, err := purego.Dlopen("/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation", purego.RTLD_GLOBAL|purego.RTLD_NOW)
		if err != nil {
			panic(err)
		}
		mainQ, err = purego.Dlsym(sys, "_dispatch_main_q")
		if err != nil {
			panic(err)
		}
		purego.RegisterLibFunc(&dispatchAsyc, sys, "dispatch_async_f")
		purego.RegisterLibFunc(&cfRunLoopRun, cf, "CFRunLoopRun")
		trampoline = purego.NewCallback(func(ctx uintptr) {
			mu.Lock()
			fn := tasks[ctx]
			delete(tasks, ctx)
			mu.Unlock()
			if fn != nil {
				fn()
			}
		})
	})
}

// RunOnMain queues fn on the main dispatch queue.
func RunOnMain(fn func()) {
	setup()
	mu.Lock()
	next++
	k := next
	tasks[k] = fn
	mu.Unlock()
	dispatchAsyc(mainQ, k, trampoline)
}

// Run runs work on a goroutine and the main run loop on the calling (main) goroutine; never returns.
// work should call os.Exit when done.
func Run(work func()) {
	setup()
	go work()
	cfRunLoopRun()
}
