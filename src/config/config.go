package config

import (
    "encoding/json"
    "fmt"
    "reflect"
)

var interfaceSlice []interface{}
var interfaceType = reflect.TypeOf(interfaceSlice).Elem()

type subParser struct {
    def reflect.Value
}

type Parser struct {
    def reflect.Value
    typ reflect.Type // Struct that has its types converted to interfaces
    sub []*subParser
    vf  map[uintptr] reflect.Value
}

// Validators

func NewParser(cfg interface{}) (*Parser, error) {

    // Ensure cfg is struct
    // cfg needs to be a pointer in order to be addressable value
    if isPtrToStruct(cfg) == false {
        return nil, fmt.Errorf("The given parameter is not a pointer to struct")
    }

    // Struct to interface struct
    def := reflect.ValueOf(cfg).Elem()
    typ := fieldsToInterface(def.Type())
    
    // Return
    return &Parser{
        def: def,
        typ: typ,
        sub: make([]*subParser, 0),
        vf: make(map[uintptr] reflect.Value),
    }, nil

}

// Make fiels whose zero value is not nil into interfaces
func fieldsToInterface(typ reflect.Type) reflect.Type {

    nf     := typ.NumField()
    fields := make([]reflect.StructField, nf)
    
    for i := 0; i < nf; i++ {
        fields[i] = typ.Field(i)
        switch fields[i].Type.Kind() {
        case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
            reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16,
            reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64,
            reflect.String:
            fields[i].Type = interfaceType
        case reflect.Slice:
            // Check for struct
            if fields[i].Type.Elem().Kind() == reflect.Struct {
                fields[i].Type = reflect.SliceOf(fieldsToInterface(fields[i].Type.Elem()))
            }
        case reflect.Map:
            // Check for struct
            if fields[i].Type.Elem().Kind() == reflect.Struct {
                fields[i].Type = reflect.MapOf(
                    fields[i].Type.Key(), fieldsToInterface(fields[i].Type.Elem()),
                )
            }
        case reflect.Struct:
            // Recursive
            fields[i].Type = fieldsToInterface(fields[i].Type)
        default:
        }
    }

    return reflect.StructOf(fields)

}

func isPtrToStruct(pstr interface{}) bool {
    rv := reflect.ValueOf(pstr)
    return rv.Kind() == reflect.Ptr && rv.Elem().Kind() == reflect.Struct
}

// PARSER ---

func(p *Parser) Parse(data []byte, pstr interface{}) (err error) {

    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("%v", r)
        }
    }()

    if isPtrToStruct(pstr) == false {
        return fmt.Errorf("The given parameter is not a pointer to struct")
    }

    // Ensure same type as default config
    rv := reflect.ValueOf(pstr)
    el := rv.Elem()
    if el.Type() != p.def.Type() {
        return fmt.Errorf("The given struct is not the same type as the default configuration")
    }

    // Make new struct
    pa  := reflect.New(p.typ)
    a   := pa.Elem()
    err  = json.Unmarshal(data, pa.Interface())
    if err != nil {
        return err
    }

    // Default
    pb := reflect.New(p.def.Type())
    b  := pb.Elem()

    // Deep fill nil
    p.deepFillNil(p.def, a, b)

    // Assign
    el.Set(b)

    return nil

}

func(p *Parser) deepFillNil(def, a, b reflect.Value) { // a => b

    for i := 0; i < a.NumField(); i++ {

        dv := def.Field(i)
        av := a.Field(i)
        bv := b.Field(i)

        if bv.Type().Kind() == reflect.Struct {
            p.deepFillNil(dv, av, bv)
        } else {

            if av.IsNil() {

                // Default values
                switch bv.Type().Kind() {
                case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
                    reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16,
                    reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
                    // Convert numbers
                    bv.Set(dv.Convert(bv.Type()))
                // case reflect.Bool: passthrough
                // case reflect.String: passthrough
                default:
                    bv.Set(dv)
                }

            } else {
                
                switch bv.Type().Kind() {
                case reflect.Bool:
                    b := av.Interface().(bool)
                    bv.SetBool(b)

                case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
                    reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16,
                    reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:

                    // All numbers in elNew is float64 as json decoded it
                    rf64 := reflect.ValueOf(av.Interface().(float64))
                    bv.Set(rf64.Convert(bv.Type()))

                case reflect.String:
                    s := av.Interface().(string)
                    bv.SetString(s)

                case reflect.Slice, reflect.Map:

                    bvElTyp := bv.Type().Elem()

                    // Check for sub parser
                    var sub *subParser
                    for _, sp := range p.sub {
                        // Compare element type
                        if bvElTyp == sp.def.Type() {
                            sub = sp
                            break
                        }
                    }

                    if sub == nil {
                        bv.Set(av)
                    } else {
                        
                        // Deep copy each element
                        switch bv.Type().Kind() {
                        case reflect.Slice:
                            bv.Set(reflect.MakeSlice(bv.Type(), 0, 0))
                            for k := 0; k < av.Len(); k++ {
                                subAv  := av.Index(k)
                                psubBv := reflect.New(bvElTyp)
                                subBv  := psubBv.Elem()
                                p.deepFillNil(sub.def, subAv, subBv)
                                bv.Set(reflect.Append(bv, subBv))
                            }
                        case reflect.Map:
                            bv.Set(reflect.MakeMap(bv.Type()))
                            keys := av.MapKeys()
                            for k := 0; k < len(keys); k++ {
                                key    := keys[k]
                                subAv  := av.MapIndex(key)
                                psubBv := reflect.New(bvElTyp)
                                subBv  := psubBv.Elem()
                                p.deepFillNil(sub.def, subAv, subBv)
                                bv.SetMapIndex(key, subBv)
                            }
                        }

                    }

                default:
                    bv.Set(av)
                }

            }

            // Validate
            rvf, ok := p.vf[dv.Addr().Pointer()]
            if ok {
                out   := rvf.Call([]reflect.Value{bv})[0]
                valid := out.Interface().(bool)
                if !valid {
                    panic(fmt.Errorf(
                        "%s.%s has an invalid value of %v",
                        b.Type().Name(), b.Type().Field(i).Name,
                        bv,
                    ))
                }
            }

        }
    }

}

func(p *Parser) Validator(ptr, vf interface{}) error {

    rptr := reflect.ValueOf(ptr)
    rel  := rptr.Elem()
    rvf  := reflect.ValueOf(vf)

    // Ensure function is func(type) bool
    if rvf.Type().NumIn() != 1 {
        return fmt.Errorf("Given function has invalid parameter count")
    }
    if rvf.Type().In(0) != rel.Type() {
        return fmt.Errorf(
            "Wrong parameter type, %v, for validator function for %v",
            rvf.Type().In(0), rel.Type(),
        )
    }
    if rvf.Type().NumOut() != 1 || rvf.Type().Out(0).Kind() != reflect.Bool {
        return fmt.Errorf("Wrong return type for validator function")
    }

    // Assign
    p.vf[rptr.Pointer()] = rvf

    return nil

}

// Child parser
func(p *Parser) SubParsers(cfgs ...interface{}) (err error) {

    // Add parsers for structs inside array, map, or slice

    for _, cfg := range cfgs {

        // Ensure cfg is struct
        if isPtrToStruct(cfg) == false {
            return fmt.Errorf("One of the given parameters is not a pointer to struct")
        }

        // Struct to interface struct
        def   := reflect.ValueOf(cfg).Elem()
        p.sub  = append(p.sub, &subParser{
            def: def,
        })

    }

    return nil

}