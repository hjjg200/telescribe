package config

import (
    "encoding/json"
    "fmt"
    "reflect"
)

var interfaceSlice []interface{}
var interfaceType = reflect.TypeOf(interfaceSlice).Elem()

type Parser struct {
    def reflect.Value
    typ reflect.Type // Struct that has its types converted to interfaces
    sub []reflect.Value
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
        sub: make([]reflect.Value, 0),
        vf: make(map[uintptr] reflect.Value),
    }, nil

}

// Make fiels whose zero value is not nil into interfaces
func fieldsToInterface(typ reflect.Type) reflect.Type {

    nf     := typ.NumField()
    fields := make([]reflect.StructField, 0)
    
    for i := 0; i < nf; i++ {

        field := typ.Field(i)

        // Check if exported
        first := field.Name[0]
        if first < 'A' || first > 'Z' {
            continue
        }

        switch field.Type.Kind() {
        case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
            reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16,
            reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64,
            reflect.String:
            field.Type = interfaceType
        case reflect.Slice:
            // Check for struct
            if field.Type.Elem().Kind() == reflect.Struct {
                field.Type = reflect.SliceOf(fieldsToInterface(field.Type.Elem()))
            }
        case reflect.Map:
            // Check for struct
            if field.Type.Elem().Kind() == reflect.Struct {
                field.Type = reflect.MapOf(
                    field.Type.Key(), fieldsToInterface(field.Type.Elem()),
                )
            }
        case reflect.Struct:
            // Recursive
            field.Type = fieldsToInterface(field.Type)
        default:
        }

        fields = append(fields, field)

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

        // Check if exported
        first := b.Type().Field(i).Name[0]
        if first < 'A' || first > 'Z' {
            continue
        }

        if bv.Type().Kind() == reflect.Struct {
            p.deepFillNil(dv, av, bv)
        } else {

            if av.IsNil() {

                // Default values
                bv.Set(dv)

            } else {
                
                // Convert interfaces
                switch bv.Type().Kind() {
                case reflect.Bool:
            
                    v := av.Interface().(bool)
                    bv.SetBool(v)
            
                case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
                    reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16,
                    reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
            
                    // All numbers in a is float64 as json decoded it
                    rf64 := reflect.ValueOf(av.Interface().(float64))
                    bv.Set(rf64.Convert(bv.Type()))
            
                case reflect.String:
            
                    s := av.Interface().(string)
                    bv.SetString(s)

                case reflect.Slice, reflect.Map:

                    bvElTyp := bv.Type().Elem()

                    if bvElTyp.Kind() == reflect.Struct {

                        // Check for sub def
                        // Zero value for bv elem type is fallback in case of no sub default
                        sub := reflect.New(bvElTyp).Elem()
                        for _, each := range p.sub {
                            // Compare element type
                            if bvElTyp == each.Type() {
                                sub = each
                                break
                            }
                        }

                        // Deep copy each element
                        switch bv.Type().Kind() {
                        case reflect.Slice:
                            bv.Set(reflect.MakeSlice(bv.Type(), 0, 0))
                            for k := 0; k < av.Len(); k++ {
                                subAv := av.Index(k)
                                subBv := reflect.New(bvElTyp).Elem()
                                p.deepFillNil(sub, subAv, subBv)
                                bv.Set(reflect.Append(bv, subBv))
                            }
                        case reflect.Map:
                            bv.Set(reflect.MakeMap(bv.Type()))
                            keys := av.MapKeys()
                            for _, key := range keys {
                                subAv := av.MapIndex(key)
                                subBv := reflect.New(bvElTyp).Elem()
                                p.deepFillNil(sub, subAv, subBv)
                                bv.SetMapIndex(key, subBv)
                            }
                        }

                    } else {

                        bv.Set(av)

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
        p.sub  = append(p.sub, def)

    }

    return nil

}