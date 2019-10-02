package monitor

import (
    "bytes"
    "io"
    "os"
    "regexp"
)

func readFile(path string) (string, error) {
    
    f, err := os.Open(path)
    if err != nil {
        return "", err
    }
    defer f.Close()

    buf := bytes.NewBuffer(nil)
    _, err = io.Copy(buf, f)
    if err != nil {
        return "", err
    }

    // For convenience, returns as string
    return string(buf.Bytes()), nil

}

var whitespaceRegexp = regexp.MustCompile("\\s+")
func splitWhitespace(str string) []string {
    return whitespaceRegexp.Split(str, -1)
}