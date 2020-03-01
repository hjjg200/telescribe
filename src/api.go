package main

import (
    "encoding/json"
    "fmt"
    "io"
    "strings"
    "time"

    . "github.com/hjjg200/go-act"
)

/*

HTTP -> HttpRequest -> APIRequest -> f(APIAction) -> Data and Error -> APIRequest.Pop(Data, Error)

*/

type APIResponse struct {
    Data map[string] interface{} `json:"data,omitempty"`
    Error APIError `json:"error,omitempty"`
    Meta APIMeta `json:"meta"`
}

type APIError interface {
    Code() int
    Message() string
}

type apiError struct {
    code int `json:"code"`
    message string `json:"string"`
}

var APIErrorBadRequest = apiError{ 101, "Bad request" }
var APIErrorJSON = apiError{ 102, "JSON error" }

func NewAPIError(code int, msg string) APIError {
    return apiError{ code, msg }
}
func(ae apiError) Code() int { return ae.code }
func(ae apiError) Message() string { return ae.message }
func(ae apiError) Error() string { return fmt.Sprintf("%d: %s", ae.code, ae.message) }

type APIData map[string] interface{}

type APIMeta struct {
    Timestamp int64 `json:"timestamp"`
    Request string `json:"request"`
}

type APIRequest struct {
    RouterName string
    ActionName string
    Executor string
    Method string
    Args []string
    hrq HttpRequest
}

func(arq *APIRequest) makeMeta() APIMeta {
    return APIMeta{
        Timestamp: time.Now().Unix(),
        Request: arq.ToString(),
    }
}

func(arq *APIRequest) ToString() string {
    return fmt.Sprintf(
        "%s %s@%s/%s/%s",
        arq.Method, arq.Executor, arq.RouterName, arq.ActionName,
        strings.Join(arq.Args, "/"),
    )
}

func(arq *APIRequest) json(d map[string] interface{}, e APIError) {
    w := arq.hrq.Writer
    w.Header().Set("Content-Type", "application/json")
    enc := json.NewEncoder(w)
    enc.SetIndent("", "  ")
    rsp := APIResponse{
        Data: d,
        Error: e,
        Meta: arq.makeMeta(),
    }
    if enc.Encode(rsp) != nil {
        w.WriteHeader(500)
        w.Write([]byte("{}"))
    }
}

func(arq *APIRequest) Data(d map[string] interface{}) {
    arq.json(d, nil)
}

func(arq *APIRequest) Error(err APIError) {
    arq.json(nil, err)
}

func(arq *APIRequest) ContentType(ct string) {
    w := arq.hrq.Writer
    w.Header().Set("Content-Type", ct)
}

func(arq *APIRequest) ReadFrom(r io.Reader) {
    w := arq.hrq.Writer
    io.Copy(w, r)
}

/**

API Request



**/


type APIAction func(APIRequest)

type APIRouter struct {
    name string
    actions map[string] map[string] APIAction
}

func NewAPIRouter(n string) *APIRouter {
    return &APIRouter{
        name: n,
        actions: make(map[string] map[string] APIAction),
    }
}

func(ar *APIRouter) Serve(req HttpRequest) {

    uri, ok := ar.trimPrefix(req.Body.URL.Path)
    if !ok {
        req.Writer.WriteHeader(400)
        return
    }
    uri  = strings.TrimSuffix(uri, "/")
    args := strings.Split(uri, "/")
    usr, _, _ := req.Body.BasicAuth()
    arq  := APIRequest{
        RouterName: ar.name,
        ActionName: args[0],
        Executor: usr,
        Method: req.Body.Method,
        Args: args[1:],
        hrq: req,
    }

    defer func() {
        r := recover()
        if r != nil {
            arq.Error(APIErrorBadRequest)
            return
        }
    }()

    action, ok := ar.actions[arq.ActionName][arq.Method]
    Assert(ok, "Action not found")

    action(arq)
}

func(ar *APIRouter) trimPrefix(u string) (string, bool) {
    least := len(ar.name) + 3 // name + 2 slashes + at least 1 letter
    if len(u) < least { 
        return "", false
    }
    rn := u[1:least - 2] // routerName
    if rn != ar.name {
        return "", false
    }
    return u[least - 1:], true // subtract 1 to include the first letter
}

func(ar *APIRouter) Get(n string, f APIAction) { ar.add("GET", n, f) }
func(ar *APIRouter) Head(n string, f APIAction) { ar.add("HEAD", n, f) }
func(ar *APIRouter) Post(n string, f APIAction) { ar.add("POST", n, f) }
func(ar *APIRouter) Put(n string, f APIAction) { ar.add("PUT", n, f) }
func(ar *APIRouter) Delete(n string, f APIAction) { ar.add("DELETE", n, f) }
func(ar *APIRouter) Options(n string, f APIAction) { ar.add("OPTIONS", n, f) }
func(ar *APIRouter) Patch(n string, f APIAction) { ar.add("PATCH", n, f) }
func(ar *APIRouter) add(m string, n string, f APIAction) {
    
    if _, ok := ar.actions[n]; !ok {
        ar.actions[n] = make(map[string] APIAction)
    }
    m = strings.ToUpper(m) // Uppercase
    ar.actions[n][m] = f

}