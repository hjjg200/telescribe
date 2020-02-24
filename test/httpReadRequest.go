package main

import (
    "bufio"
    "bytes"
    "fmt"
    "io/ioutil"
    "net"
    "net/http"
)

func main() {

    fmt.Println("HTTP Read Request Test")

    mainLn, _ := net.Listen("tcp", "0.0.0.0:8081")
    for {
        conn, _ := mainLn.Accept()
        go func() {
            fmt.Println("New connection")
            rd := bufio.NewReader(conn)
            for {
                req, err := http.ReadRequest(rd)
                if err != nil {
                    fmt.Println("err:", err)
                    return
                }
                fmt.Println(req.RemoteAddr, req.URL.Path)
                body := bytes.NewReader([]byte(req.URL.Path))
                rsp := http.Response{
                    Status: "200 OK",
                    StatusCode: 200,
                    Proto: "HTTP/1.1", // 1.0 appears to be not supporting keeping connections open
                    ProtoMajor: 1,
                    ProtoMinor: 1,
                    Close: false, // Do not close the connection
                    Body: ioutil.NopCloser(body),
                    ContentLength: body.Size(),
                }
                rsp.Write(conn)
            }
        }()
    }

}
