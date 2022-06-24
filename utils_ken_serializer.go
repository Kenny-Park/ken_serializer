package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"reflect"
	"strconv"
)

type KenSerializer struct{}

const INTEGER = 8

// convert int to bytes
func (c KenSerializer) toInt(i int64) []byte {
	bs := make([]byte, INTEGER)
	binary.BigEndian.PutUint64(bs, uint64(i))
	return bs
}

// convert bytes to int
func (c KenSerializer) fromInt(f []byte) int {
	r := binary.BigEndian.Uint64(f)
	return int(r)
}

// convert float to bytes
func (c KenSerializer) toFloat(f float64) []byte {
	var buf bytes.Buffer
	err := binary.Write(&buf, binary.BigEndian, f)
	if err != nil {
		fmt.Println("binary.Write failed:", err)
	}
	return buf.Bytes()
}

// convert bytes to float
func (c KenSerializer) fromFloat(f []byte) float64 {
	bits := binary.BigEndian.Uint64(f)
	r := math.Float64frombits(bits)
	return r
}

func (c KenSerializer) ConvertInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

func (c KenSerializer) typeCheckForGet(s reflect.Value, size int) []byte {
	if !s.IsValid() {
		return nil
	}

	var bs []byte
	// 변수 타입으로 값 구분
	switch s.Kind() {
	case reflect.Func:
		return nil
	// string kind 일 경우
	case reflect.String:
		bs = make([]byte, size)
		bs1 := []byte(s.String())
		copy(bs, bs1)

	// float kind 일 경우
	case reflect.Float64:
		bs = c.toFloat(s.Float())

	case reflect.Float32:
		bs = c.toFloat(s.Float())

	// int kind일 경우
	case reflect.Int:
		bs = c.toInt(s.Int())
	case reflect.Int32:
		bs = c.toInt(s.Int())
	case reflect.Int64:
		bs = c.toInt(s.Int())

	case reflect.Bool:
		if s.Bool() == false {
			bs = []byte{byte(0)}
		} else {
			bs = []byte{byte(1)}
		}

	// slice kind일 경우
	case reflect.Slice:
		// []byte 일때는 바로 출력해줌
		if s.Type() == reflect.TypeOf([]byte{}) {
			if s.IsNil() || s.Len() == 0 {
				ms := make([]byte, size, size)
				return ms
			} else {
				b := make([]byte, size)
				copy(b, s.Interface().([]byte))
				return b
			}
		}

		arrItem := reflect.ValueOf(s.Interface())

		// 배열의 length를 취득한다.
		bs = append(bs, c.toInt(int64(arrItem.Len()))...)

		for j := 0; j < arrItem.Len(); j++ {
			if arrItem.Index(j).Kind() == reflect.Pointer {
				bs = append(bs, c.typeCheckForGet(reflect.Indirect(arrItem.Index(j)), size)...)
			} else {
				bs = append(bs, c.typeCheckForGet(arrItem.Index(j), size)...)
			}
		}

	// pointer일 경우
	case reflect.Pointer:
		dd := reflect.Indirect(s).Interface()
		bs = append(bs, KenSerializer{}.ToByte(dd)...)

	// struct kind일 경우
	case reflect.Struct:
		dd := s.Interface()
		bs = append(bs, KenSerializer{}.ToByte(dd)...)

	}
	return bs
}

func (c KenSerializer) typeCheckForSet(vo interface{}, n []byte, size int) []byte {

	s := vo.(reflect.Value)
	if !s.CanSet() || !s.IsValid() {
		return nil
	}

	// 변수 타입으로 값 구분
	switch s.Kind() {
	case reflect.Func:
		return nil
	// string kind 일 경우
	case reflect.String:
		a := n[:size]
		var an []byte

		for _, item := range a {
			if byte(0) != item {
				an = append(an, item)
			}
		}
		s.SetString(string(an))
		n = n[size:]

	// float kind 일 경우
	case reflect.Float64:
		s.SetFloat(c.fromFloat(n[:int(INTEGER)]))
		n = n[int(INTEGER):]

	case reflect.Float32:
		s.SetFloat(c.fromFloat(n[:int(INTEGER)]))
		n = n[int(INTEGER):]
	// int kind일 경우
	case reflect.Int:
		s.SetInt(int64(c.fromInt(n[:int(INTEGER)])))
		n = n[int(INTEGER):]
	case reflect.Int32:
		s.SetInt(int64(c.fromInt(n[:int(INTEGER)])))
		n = n[int(INTEGER):]
	case reflect.Int64:
		s.SetInt(int64(c.fromInt(n[:int(INTEGER)])))
		n = n[int(INTEGER):]

	case reflect.Bool:
		s.SetBool(func() bool {
			if n[0] == byte(0) {
				return false
			} else {
				return true
			}
		}())
		n = n[1:]

	// slice kind일 경우
	case reflect.Slice:
		// []byte 일때는 바로 출력해줌
		if s.Type() == reflect.TypeOf([]byte{}) {
			b := make([]byte, size)
			copy(b, n[:size])
			s.SetBytes(b)
			return n[size:]
		}

		arrSize := c.fromInt(n[:int(INTEGER)])
		n = n[int(INTEGER):]

		if arrSize == 0 {
			return n
		}
		ns := reflect.MakeSlice(reflect.Indirect(s).Type(), int(arrSize), int(arrSize))
		s.Set(ns)

		for j := 0; j < s.Len(); j++ {
			if s.Index(j).Kind() == reflect.Pointer {
				n = c.ToStruct(n, s.Index(j).Elem())
			} else if s.Index(j).Kind() == reflect.Struct {
				n = c.ToStruct(n, s.Index(j))
			} else {
				n = c.typeCheckForSet(s.Index(j), n, int(size))
			}
		}

	// pointer일 경우
	case reflect.Pointer:
		n = c.ToStruct(n, s.Elem())
	case reflect.Struct:
		n = c.ToStruct(n, s)
	}

	return n
}

func (c KenSerializer) ToStruct(b []byte, o interface{}) []byte {

	var vo reflect.Value
	var t reflect.Type
	if reflect.TypeOf(o) == reflect.TypeOf(reflect.Value{}) {
		vo = o.(reflect.Value)
	} else {
		vo = reflect.ValueOf(o)
	}

	t = reflect.Indirect(vo).Type()

	if vo.Kind() == reflect.Pointer {
		for i := 0; i < vo.Elem().NumField(); i++ {
			if t.Field(i).Tag.Get("flag") == "N" {
				continue
			}
			kv := vo.Elem().Field(i)
			size := c.ConvertInt(t.Field(i).Tag.Get("size"))
			if reflect.ValueOf(kv).IsValid() {
				b = c.typeCheckForSet(kv, b, size)
			}
		}
	} else {
		for i := 0; i < vo.NumField(); i++ {
			if t.Field(i).Tag.Get("flag") == "N" {
				continue
			}
			kv := vo.Field(i)
			size := c.ConvertInt(t.Field(i).Tag.Get("size"))
			if reflect.ValueOf(kv).IsValid() {
				b = c.typeCheckForSet(kv, b, size)
			}
		}
	}
	return b
}

func (c KenSerializer) ToByte(o interface{}) []byte {
	var bs []byte
	var result []byte

	s := reflect.ValueOf(o)
	t := reflect.TypeOf(o)

	fun := func(s reflect.Value, t reflect.Type) {
		for i := 0; i < s.NumField(); i++ {
			if t.Field(i).Tag.Get("flag") == "N" {
				continue
			}
			if s.Field(i).IsValid() {
				size := c.ConvertInt(t.Field(i).Tag.Get("size"))
				bs = c.typeCheckForGet(s.Field(i), size)
				result = append(result, bs...)
			}
		}
	}

	if s.Kind() == reflect.Slice {
		item := reflect.ValueOf(s)
		for j := 0; j < item.Len(); j++ {
			bs = c.ToByte(item.Index(j))
			result = append(result, bs...)
		}
	} else if s.Kind() == reflect.Struct {
		fun(s, t)
	} else if s.Kind() == reflect.Pointer {
		fun(reflect.Indirect(s), reflect.TypeOf(reflect.Indirect(s)))
	} else {
		log.Println("can`t convert struct to bytes")
	}

	return result
}
