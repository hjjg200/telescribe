package log

import (
    "compress/gzip"
    "fmt"
    "io"
    "io/ioutil"
    "os"
    "time"

    "../util"

    . "github.com/hjjg200/go-act"
)

var FileDir = "./logs"
var FileTimeFormat = "2006Jan02"
var fileExt = ".log"
var fileArchiveExt = ".gz"

type File struct {
    f *os.File
}

func NewFile(prefix string) (lf *File, err error) {
    defer Catch(&err)

    Try(util.EnsureDirectory(FileDir))

    // Path
    name := newFileName(prefix)
    path := FileDir + "/" + name

    // Open
    f, err := os.OpenFile(path, os.O_CREATE | os.O_WRONLY, 0600)
    Try(err)

    // File
    lf = &File{f: f}

    return lf, nil
}

func(lf *File) Write(p []byte) (int, error) {
    return lf.f.Write(p)
}

func(lf *File) Close() (err error) {
    defer Catch(&err)

    // Reopen for read
    fn := lf.f.Name()
    Try(lf.f.Close())
    lf.f, err = os.OpenFile(fn, os.O_RDONLY, 0600)
    Try(err)

    // Open gzip
    // -> os.File.Name() gives you name as presented to Open
    gzf, err := os.OpenFile(fn + fileArchiveExt, os.O_CREATE | os.O_WRONLY, 0600)
    Try(err)

    gz := gzip.NewWriter(gzf)

    // Write to gzip
    _, err = io.Copy(gz, lf.f)
    Try(err)
    Try(gz.Close())
    Try(gzf.Close())

    // Remove log
    Try(lf.f.Close())
    Try(os.Remove(fn))

    return nil
}

func formatFileName(prefix string, idx int) string {
    ts   := time.Now().Format(FileTimeFormat)
    name := fmt.Sprintf("%s.%s.%d%s", prefix, ts, idx, fileExt)
    return name
}

func newFileName(prefix string) string {
    fis, err := ioutil.ReadDir(FileDir)
    Try(err)

    count := 0 // Count for log files of the same day
    name  := formatFileName(prefix, count)
    for _, fi := range fis {
        // Check if the log file of the same name already exists
        if fi.Name() == name || fi.Name() == name + fileArchiveExt {
            count++
            name = formatFileName(prefix, count)
        }
    }

    return name
}