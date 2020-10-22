package monitor

// #include <sys/statvfs.h>
import "C"

// statfs() is OS-specific
// statvfs() is posix-conforming
// https://stackoverflow.com/questions/1653163/difference-between-statvfs-and-statfs-system-calls

import (
    "fmt"
    "reflect"
)

type Statvfs_t struct {
    Bsize   uint64
    Frsize  uint64
    Blocks  uint64
    Bfree   uint64
    Bavail  uint64
    Files   uint64
    Ffree   uint64
    Favail  uint64
    Fsid    uint64
    Flag    uint64
    Namemax uint64
}

func c_PROPAGATE_ALL_ONES(in interface{}) uint64 {

    all := false
    rv  := reflect.ValueOf(in)
    sz  := rv.Type().Bits() / 8
    val := rv.Uint() // uint64

    switch sz {
    case 8: all = ^val == 0
    case 4: all = ^uint32(val) == 0
    case 2: all = ^uint16(val) == 0
    case 1: all = ^uint8(val) == 0
    }

    if all {return ^uint64(0)}
    return val

}

func Statvfs(path string, buf *Statvfs_t) (err error) {

    c_stat := C.struct_statvfs{}
    c_path := C.CString(path)
    result := C.statvfs(c_path, &c_stat)

    if result != 0 {
        return fmt.Errorf("Failed to call statvfs")
    }

    *buf = Statvfs_t{
        Bsize:   c_PROPAGATE_ALL_ONES(c_stat.f_bsize),
        Frsize:  c_PROPAGATE_ALL_ONES(c_stat.f_frsize),
        Blocks:  c_PROPAGATE_ALL_ONES(c_stat.f_blocks),
        Bfree:   c_PROPAGATE_ALL_ONES(c_stat.f_bfree),
        Bavail:  c_PROPAGATE_ALL_ONES(c_stat.f_bavail),
        Files:   c_PROPAGATE_ALL_ONES(c_stat.f_files),
        Ffree:   c_PROPAGATE_ALL_ONES(c_stat.f_ffree),
        Favail:  c_PROPAGATE_ALL_ONES(c_stat.f_favail),
        Fsid:    c_PROPAGATE_ALL_ONES(c_stat.f_fsid),
        Flag:    c_PROPAGATE_ALL_ONES(c_stat.f_flag),
        Namemax: c_PROPAGATE_ALL_ONES(c_stat.f_namemax),
    }

    fmt.Println(path, c_stat)

    return nil

}