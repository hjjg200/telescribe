package util

import (
    "fmt"
    "os"
)

func EnsureDirectory(p string) error {
    st, err := os.Stat(p)
    if err != nil {
        if os.IsNotExist(err) {
            err = os.MkdirAll(p, 0755)
            if err != nil {
                return err
            }
        }
        return err
    }
    if !st.IsDir() {
        return fmt.Errorf("Path does not resolve to a directory!")
    }
    return nil
}