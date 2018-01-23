// Copyright (C) 2015  Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package test

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"strconv"

	"github.com/aristanetworks/goarista/areflect"
)

// PrettyPrint tries to display a human readable version of an interface
func PrettyPrint(v interface{}) string {
	return PrettyPrintWithDepth(v, prettyPrintDepth)
}

// PrettyPrintWithDepth tries to display a human readable version of an interface
// and allows to define the depth of the print
func PrettyPrintWithDepth(v interface{}, depth int) string {
	return prettyPrint(reflect.ValueOf(v), ptrSet{}, depth)
}

var prettyPrintDepth = 8

func init() {
	d := os.Getenv("PPDEPTH")
	if d, ok := strconv.Atoi(d); ok == nil && d >= 0 {
		prettyPrintDepth = d
	}
}

type ptrSet map[uintptr]struct{}

func prettyPrint(v reflect.Value, done ptrSet, depth int) string {
	return prettyPrintWithType(v, done, depth, true)
}

func prettyPrintWithType(v reflect.Value, done ptrSet, depth int, showType bool) string {
	if depth < 0 {
		return "<max_depth>"
	}
	switch v.Kind() {
	case reflect.Invalid:
		return "nil"
	case reflect.Bool:
		return fmt.Sprintf("%t", v.Bool())
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Float32, reflect.Float64,
		reflect.Int, reflect.Uint, reflect.Uintptr,
		reflect.Complex64, reflect.Complex128:
		i := areflect.ForceExport(v).Interface()
		if showType {
			return fmt.Sprintf("%s(%v)", v.Type().Name(), i)
		}
		return fmt.Sprintf("%v", i)
	case reflect.String:
		return fmt.Sprintf("%q", v.String())
	case reflect.Ptr:
		return "*" + prettyPrintWithType(v.Elem(), done, depth-1, showType)
	case reflect.Interface:
		return prettyPrintWithType(v.Elem(), done, depth-1, showType)
	case reflect.Map:
		var r []byte
		r = append(r, []byte(v.Type().String())...)
		r = append(r, '{')
		var elems mapEntries
		for _, k := range v.MapKeys() {
			elem := &mapEntry{
				k: prettyPrint(k, done, depth-1),
				v: prettyPrint(v.MapIndex(k), done, depth-1),
			}
			elems.entries = append(elems.entries, elem)
		}
		sort.Sort(&elems)
		for i, e := range elems.entries {
			if i > 0 {
				r = append(r, []byte(", ")...)
			}
			r = append(r, []byte(e.k)...)
			r = append(r, ':')
			r = append(r, []byte(e.v)...)
		}
		r = append(r, '}')
		return string(r)
	case reflect.Struct:
		// Circular dependency?
		if v.CanAddr() {
			ptr := v.UnsafeAddr()
			if _, ok := done[ptr]; ok {
				return fmt.Sprintf("%s{<circular dependency>}", v.Type().String())
			}
			done[ptr] = struct{}{}
		}
		var r []byte
		r = append(r, []byte(v.Type().String())...)
		r = append(r, '{')
		for i := 0; i < v.NumField(); i++ {
			if i > 0 {
				r = append(r, []byte(", ")...)
			}
			sf := v.Type().Field(i)
			r = append(r, sf.Name...)
			r = append(r, ':')
			r = append(r, prettyPrint(v.Field(i), done, depth-1)...)
		}
		r = append(r, '}')
		return string(r)
	case reflect.Chan:
		var ptr, bufsize string
		if v.Pointer() == 0 {
			ptr = "nil"
		} else {
			ptr = fmt.Sprintf("0x%x", v.Pointer())
		}
		if v.Cap() > 0 {
			bufsize = fmt.Sprintf("[%d]", v.Cap())
		}
		return fmt.Sprintf("(%s)(%s)%s", v.Type().String(), ptr, bufsize)
	case reflect.Func:
		return "func(...)"
	case reflect.Array, reflect.Slice:
		l := v.Len()
		var r []byte
		if v.Type().Elem().Kind() == reflect.Uint8 && v.Kind() != reflect.Array {
			b := areflect.ForceExport(v).Interface().([]byte)
			r = append(r, []byte(`[]byte(`)...)
			if b == nil {
				r = append(r, []byte("nil")...)
			} else {
				r = append(r, []byte(fmt.Sprintf("%q", b))...)
			}
			r = append(r, ')')
			return string(r)
		}
		r = append(r, []byte(v.Type().String())...)
		r = append(r, '{')
		for i := 0; i < l; i++ {
			if i > 0 {
				r = append(r, []byte(", ")...)
			}
			r = append(r, prettyPrintWithType(v.Index(i), done, depth-1, false)...)
		}
		r = append(r, '}')
		return string(r)
	case reflect.UnsafePointer:
		var ptr string
		if v.Pointer() == 0 {
			ptr = "nil"
		} else {
			ptr = fmt.Sprintf("0x%x", v.Pointer())
		}
		if showType {
			ptr = fmt.Sprintf("(unsafe.Pointer)(%s)", ptr)
		}
		return ptr
	default:
		panic(fmt.Errorf("Unhandled kind of reflect.Value: %v", v.Kind()))
	}
}
