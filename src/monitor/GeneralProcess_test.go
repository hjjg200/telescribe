package monitor

import (
    "fmt"
    "testing"
    "time"
)

func TestViReadBytes(t *testing.T) {
    
    for {

        fmt.Println("Read and writes of vi:")
        fmt.Print("Read bytes :"); fmt.Println(GetProcessReadBytes("vi"))
        fmt.Print("Write bytes:"); fmt.Println(GetProcessWriteBytes("vi"))

        time.Sleep(3 * time.Second)

    }

}