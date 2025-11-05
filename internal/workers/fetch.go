//go:build js && wasm
// +build js,wasm

package workers

import (
	"syscall/js"
)

func init() {
	patchFetch()
}

func patchFetch() {
	global := js.Global()
	fetchVal := global.Get("fetch")
	if fetchVal.IsUndefined() {
		return
	}

	originalFetch := fetchVal
	globalThis := global.Get("globalThis")
	if globalThis.IsUndefined() {
		globalThis = global
	}

	wrappedFetch := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		callArgs := make([]interface{}, len(args)+1)
		callArgs[0] = globalThis
		for i, arg := range args {
			callArgs[i+1] = arg
		}
		return originalFetch.Call("call", callArgs...)
	})

	global.Set("fetch", wrappedFetch)
}
