package main

// #include <unistd.h>
import "C"

import (
    "fmt"
)

func main() {
    fmt.Println(
        C.sysconf(C._SC_CLK_TCK),
    )
}