package log

import (
    "regexp"
)

type Filter struct {
    rgx *regexp.Regexp
    cache map[string] bool
}

func NewFilter(pattern string) (*Filter, error) {

    f := &Filter{}
    if pattern == "" { // Return empty filter for empty string
        return f, nil
    }

    err := f.SetRegexp(pattern)
    if err != nil {
        return nil, err
    }
    return f, nil

}

func newFilter(pattern string) *Filter {
    return &Filter{
        rgx: regexp.MustCompile(pattern),
        cache: make(map[string] bool),
    }
}

func(f *Filter) Filter(str string) bool {
    // If empty, always true
    if f.rgx == nil || f.cache == nil {
        return true
    }

    // Check for cache
    if b, ok := f.cache[str]; ok {
        return b
    }

    // Check and cache
    match := f.rgx.FindString(str)
    b := match == str
    f.cache[str] = b
    return b
}

func(f *Filter) SetRegexp(p string) error {
    rgx, err := regexp.Compile(p)
    if err != nil {
        return err
    }
    f.rgx = rgx
    f.cache = make(map[string] bool)
    return nil
}