package main

import (
    "fmt"
    "strings"
)

func main() {
    cmdline := "/bin/bash\x00-c\x00echo"

    fmt.Println(strings.Split(cmdline, "\x00"))
}