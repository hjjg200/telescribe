package monitor

// #include <sys/statvfs.h>
import "C"

// statfs() is OS-specific
// statvfs() is posix-conforming
// https://stackoverflow.com/questions/1653163/difference-between-statvfs-and-statfs-system-calls

import (
    "fmt"
)

type Statvfs_t struct {
    Bsize uint64
    Frsize uint64
    Blocks uint64
    Bfree uint64
    Bavail uint64
    Files uint64
    Ffree uint64
    Favail uint64
    Fsid uint64
    Flag uint64
    Namemax uint64
}

func Statvfs(path string, buf *Statvfs_t) (err error) {

    c_stat := C.struct_statvfs{}
    c_path := C.CString(path)
    result := C.statvfs(c_path, &c_stat)

    if result != 0 {
        return fmt.Errorf("Failed to call statvfs")
    }

    *buf = Statvfs_t{
        Bsize:   uint64(c_stat.f_bsize),
        Frsize:  uint64(c_stat.f_frsize),
        Blocks:  uint64(c_stat.f_blocks),
        Bfree:   uint64(c_stat.f_bfree),
        Bavail:  uint64(c_stat.f_bavail),
        Files:   uint64(c_stat.f_files),
        Ffree:   uint64(c_stat.f_ffree),
        Favail:  uint64(c_stat.f_favail),
        Fsid:    uint64(c_stat.f_fsid),
        Flag:    uint64(c_stat.f_flag),
        Namemax: uint64(c_stat.f_namemax),
    }

    return nil

}