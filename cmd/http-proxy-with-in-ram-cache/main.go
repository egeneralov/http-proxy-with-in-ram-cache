package main

import (
  "encoding/json"
  "flag"
  "fmt"
  "log"
  "sync"
  
  "github.com/valyala/fasthttp"
)

var (
  proxyClient = &fasthttp.HostClient{
    Addr: "cdimage.debian.org:80",
  }
  storage = make(map[string]*fasthttp.Response)
  bind    = "0.0.0.0:8080"
  rw      = &sync.RWMutex{}
)

func init() {
  flag.StringVar(&proxyClient.Addr, "addr", proxyClient.Addr, "Address of the proxy server")
  flag.StringVar(&bind, "bind", bind, "Address to bind to")
  flag.Parse()
}

func ReverseProxyHandler(ctx *fasthttp.RequestCtx) {
  req := &ctx.Request
  resp := &ctx.Response
  path := string(ctx.Request.URI().Path())
  
  if path == "/stats" {
    storageDump := make(map[string]int)
    rw.RLock()
    for k, v := range storage {
      storageDump[k] = len(v.Body())
    }
    rw.RUnlock()
    j, je := json.Marshal(storageDump)
    if je == nil {
      fmt.Fprintf(ctx, string(j))
    }
    return
  }
  
  _, exist := storage[path]
  if !exist {
    ctx.Logger().Printf("request")
    if err := proxyClient.Do(req, resp); err != nil {
      ctx.Logger().Printf("error when proxying the request: %s", err)
    }
    rw.Lock()
    storage[path] = &fasthttp.Response{}
    resp.CopyTo(storage[path])
    rw.Unlock()
  }
  rw.RLock()
  storage[path].CopyTo(resp)
  rw.RUnlock()
}

func main() {
  if err := fasthttp.ListenAndServe(bind, ReverseProxyHandler); err != nil {
    log.Fatalf("error in fasthttp server: %s", err)
  }
}
