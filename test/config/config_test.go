package config

import (
    "fmt"
    "testing"
)

func TestNewParser(t *testing.T) {
    
    type C struct {
        C1 []int
    }
    type B struct {
        B1 float64
        CV C
    }
    type A struct {
        A1 int
        A2 string
        BV B
    }
    def := A{
        1, "abc",
        B{
            1.2,
            C{
                C1: []int{1, 3},
            },
        },
    }

    p, _ := NewParser(&def)

    fmt.Println(p.typ)

}

func TestDeepFill(t *testing.T) {

    type BCfg struct {
        Work string `json:"work"`
        Money float64 `json:"money"`
    }
    type ACfg struct {
        Name string `json:"name"`
        Age int `json:"age"`
        B BCfg `json:"b"`
    }
    
    def := ACfg{
        Name: "John Doe",
        Age: 22,
        B: BCfg{
            Work: "Artist",
            Money: 77.5,
        },
    }
    data := `{
    "name": "abc",
    "b": {
        "money": 13.5
    }
}`

    parser, err := NewParser(&def)
    if err != nil {
        t.Error(err)
    }

    cfg := ACfg{}
    parser.Parse([]byte(data), &cfg)

    fmt.Println(cfg)

}

func TestValidator(t *testing.T) {

    type ACfg struct {
        Age int `json:"age"`
        Evens []int `json:"evens"`
    }
    def := ACfg{
        Age: 12,
        Evens: []int{2, 4, 6, 8, 10},
    }

    parser, err := NewParser(&def)
    if err != nil {
        t.Error(err)
    }

    parser.Validator(&def.Age, func(age int) bool {
        return age > 0 && age < 200
    })
    parser.Validator(&def.Evens, func(evens []int) bool {
        for _, e := range evens {
            if e % 2 != 0 {
                return false
            }
        }
        return true
    })

    data := `{
        "age": 4,
        "evens": [2, 4, 7]
    }`

    cfg := ACfg{}
    err = parser.Parse([]byte(data), &cfg)

    fmt.Println(err)

}