package main

import (
    "fmt"
    "path/filepath"
)

func main() {
    fmt.Println(filepath.EvalSymlinks("top"))
    fmt.Println(filepath.EvalSymlinks("bash"))
}