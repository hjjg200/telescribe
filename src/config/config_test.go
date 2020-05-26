package config

import (
    "encoding/json"
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
        Map map[int] string `json:"map"`
    }
    type ACfg struct {
        Name string `json:"name"`
        Age int `json:"age"`
        Map map[int] bool `json:"map"`
        B BCfg `json:"b"`
    }
    
    def := ACfg{
        Name: "John Doe",
        Age: 22,
        Map: map[int] bool{
            0: true, 1: false,
        },
        B: BCfg{
            Work: "Artist",
            Money: 77.5,
            Map: map[int] string{
                4: "abc", 5: "def",
            },
        },
    }
    data := `{
    "name": "abc",
    "map": {"4": false},
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

    t_PrettyPrint(cfg)

}

func TestValidator(t *testing.T) {

    type ACfg struct {
        Age int `json:"age"`
        Evens []int `json:"evens"`
        Map map[int] int `json:"map"`
    }
    def := ACfg{
        Age: 12,
        Evens: []int{2, 4, 6, 8, 10},
        Map: map[int] int{1: 1, 2: 2, 3:3},
    }

    parser, err := NewParser(&def)
    if err != nil {
        t.Error(err)
    }
    cfg := ACfg{}

    // Valid validators
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
    parser.Validator(&def.Map, func(m map[int] int) bool {
        for k, v := range m {
            if k != v {
                return false
            }
        }
        return true
    })

    // Invalid validators
    fmt.Println(parser.Validator(&def.Age, func(age string) bool {
        return age == "age"
    }))

    // Parse
    data := `{
        "age": 4,
        "evens": [2, 4, 6],
        "map": {"1": 1, "2": 2}
    }`
    fmt.Println(parser.Parse([]byte(data), &cfg))
    
    data = `{
        "age": -1
    }`
    fmt.Println(parser.Parse([]byte(data), &cfg))

    data = `{
        "evens": [2, 3, 6]
    }`
    fmt.Println(parser.Parse([]byte(data), &cfg))

    data = `{
        "map": {"1": 3, "2": 2}
    }`
    fmt.Println(parser.Parse([]byte(data), &cfg))

}

func TestChildDefaults(t *testing.T) {

    type CCfg struct {
        Apples int
        Bananas int
    }
    type BCfg struct {
        Stars int
    }
    type ACfg struct {
        Slice []BCfg
        Map map[string] BCfg
        SliceC []CCfg
    }

    def := ACfg{
        Slice: []BCfg{
            {11}, {22}, {33},
        },
        Map: map[string] BCfg{},
        SliceC: []CCfg{
            {2, 5}, {6, 6},
        },
    }

    bdef := BCfg{7}

    parser, err := NewParser(&def)
    if err != nil {
        t.Error(err)
    }
    err = parser.ChildDefaults(&bdef)
    if err != nil {
        t.Error(err)
    }

    parser.Validator(&bdef.Stars, func(i int) bool {
        return i > 0
    })

    cfg := ACfg{}

    // Invalid data
    data := `{
        "Slice": [
            {"Stars": 12}, {}, {"Stars": 55}
        ],
        "Map": {
            "a": {},
            "b": {"Stars": -2}
        }
    }`

    fmt.Println(parser.Parse([]byte(data), &cfg))
    t_PrettyPrint(cfg)
    
    // Valid data
    data = `{
        "Slice": [
            {"Stars": 21}, {}, {"Stars": 3}
        ],
        "Map": {
            "a": {},
            "b": {"Stars": 6}
        },
        "SliceC": [
            {"Apples": 5}
        ]
    }`

    fmt.Println(parser.Parse([]byte(data), &cfg))
    t_PrettyPrint(cfg)

}

func t_PrettyPrint(s interface{}) {
    d, _ := json.MarshalIndent(s, "", "  ")
    fmt.Println(string(d))
}

// UNEXPORTED field test
func TestUnexportedStruct(t *testing.T) {

    type BCfg struct {
        B1 int
    }
    type ACfg struct {
        A1 int
        A2 int
        a3 int
        b BCfg
    }

    def := ACfg{
        A1: 1,
        A2: 3,
    }

    parser, err := NewParser(&def)
    if err != nil {
        t.Error(err)
    }

    data := `{
        "A1": 51
    }`

    cfg := ACfg{}
    fmt.Println(parser.Parse([]byte(data), &cfg))

    t_PrettyPrint(cfg)

}