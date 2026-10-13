package main

import (
	"encoding/json"
	"syscall/js"

	"github.com/8dazo/valid-mail/internal/toolkit"
)

func main() {
	handler := js.FuncOf(func(_ js.Value, args []js.Value) any {
		response := toolkit.Response{}
		if len(args) != 1 || args[0].Type() != js.TypeString {
			response.Error = "Invalid tool request."
		} else {
			var request toolkit.Request
			if json.Unmarshal([]byte(args[0].String()), &request) != nil {
				response.Error = "Invalid tool request."
			} else {
				response = toolkit.Execute(request)
			}
		}
		data, err := json.Marshal(response)
		if err != nil {
			return `{"error":"Tool output could not be encoded."}`
		}
		return string(data)
	})
	js.Global().Set("__wemeGoRun", handler)
	select {}
}
