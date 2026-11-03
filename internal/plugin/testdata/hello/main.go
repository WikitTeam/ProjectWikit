package main

import (
	"encoding/json"
	"strconv"

	"github.com/extism/go-pdk"
)

//go:wasmimport extism:host/user kv_get
func kvGet(uint64) uint64

//go:wasmimport extism:host/user kv_set
func kvSet(uint64, uint64)

//go:wasmimport extism:host/user page_get
func pageGet(uint64) uint64

type moduleCall struct {
	Module string            `json:"module"`
	Params map[string]string `json:"params"`
	Body   string            `json:"body"`
	User   *struct {
		Username string `json:"username"`
	} `json:"user"`
}

func str(s string) uint64 {
	m := pdk.AllocateString(s)
	return m.Offset()
}

func read(offset uint64) string {
	if offset == 0 {
		return ""
	}
	m := pdk.FindMemory(offset)
	return string(m.ReadBytes())
}

//go:wasmexport module
func module() int32 {
	var call moduleCall
	if err := json.Unmarshal(pdk.Input(), &call); err != nil {
		pdk.SetError(err)
		return 1
	}
	switch call.Module {
	case "hello":
		who := "guest"
		if call.User != nil {
			who = call.User.Username
		}
		n, _ := strconv.Atoi(read(kvGet(str("views"))))
		n++
		kvSet(str("views"), str(strconv.Itoa(n)))
		pdk.OutputString("**Hello, " + who + "!** views " + strconv.Itoa(n) + "\n\n" + call.Body)
	case "echo":
		pdk.OutputString(call.Params["text"] + "|" + call.Body)
	case "spin":
		for {
		}
	case "pagetitle":
		raw := read(pageGet(str(call.Params["page"])))
		if raw == "" {
			pdk.OutputString("no such page")
			return 0
		}
		var page struct {
			Title  string  `json:"title"`
			Rating float64 `json:"rating"`
		}
		_ = json.Unmarshal([]byte(raw), &page)
		pdk.OutputString(page.Title + " " + strconv.FormatFloat(page.Rating, 'f', -1, 64))
	default:
		pdk.OutputString("unknown")
	}
	return 0
}

//go:wasmexport profile
func profile() int32 {
	var call struct {
		User struct {
			Username string `json:"username"`
		} `json:"user"`
	}
	_ = json.Unmarshal(pdk.Input(), &call)
	views := read(kvGet(str("views")))
	pdk.OutputString("**" + call.User.Username + "** views " + views)
	return 0
}

//go:wasmexport event
func event() int32 {
	var e struct {
		Event string `json:"event"`
	}
	_ = json.Unmarshal(pdk.Input(), &e)
	n, _ := strconv.Atoi(read(kvGet(str("event:" + e.Event))))
	kvSet(str("event:"+e.Event), str(strconv.Itoa(n+1)))
	return 0
}

func main() {}
