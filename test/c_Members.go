package main

// #include <sys/statvfs.h>
import "C"

// cgo doesn't like errno on Linux 
//https://github.com/golang/go/issues/1360

import (
    "fmt"
)

func main() {

    stat := C.struct_statvfs{}
    cs := C.CString("/")
    C.statvfs(cs, &stat)
    fmt.Println(stat.f_bsize)


}
