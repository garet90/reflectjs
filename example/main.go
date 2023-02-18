package main

import (
	reflectjs "github.com/garet90/reflectjs"
	"log"
	"syscall/js"
)

func add(a, b int) int {
	return a + b
}

func push(a []any, b any) []any {
	log.Println(a, b)
	return append(a, b)
}

func ret(a any) any {
	return a
}

func getAdder() func(int, int) int {
	return add
}

func do(a, b int, op func(int, int) int) int {
	return op(a, b)
}

type Dog struct {
	Age  int
	Name string
}

type Foo struct {
	Bar func() int `json:"bar"`
	Baz float64
	Dog
}

func dog(foo Foo) Dog {
	log.Printf("%#v\n", foo)
	return foo.Dog
}

func dual(f func() (int, int)) (int, int) {
	return f()
}

func variadicIn(v ...int) int {
	out := 0
	for _, vv := range v {
		out += vv
	}
	return out
}

func variadicOut(f func(...int) int) int {
	return f(1, 2, 3, 4, 5)
}

func bytesIn(b []byte) {
	log.Println(b)
}

func bytesOut() []byte {
	return []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
}

func main() {
	jsAdd := reflectjs.FuncOf(add)
	js.Global().Set("rjs_add", jsAdd)
	jsPush := reflectjs.FuncOf(push)
	js.Global().Set("rjs_push", jsPush)
	jsRet := reflectjs.FuncOf(ret)
	js.Global().Set("rjs_ret", jsRet)
	jsGetAdder := reflectjs.FuncOf(getAdder)
	js.Global().Set("rjs_getAdder", jsGetAdder)
	jsDo := reflectjs.FuncOf(do)
	js.Global().Set("rjs_do", jsDo)
	jsDog := reflectjs.FuncOf(dog)
	js.Global().Set("rjs_dog", jsDog)
	jsDual := reflectjs.FuncOf(dual)
	js.Global().Set("rjs_dual", jsDual)
	jsVariadicIn := reflectjs.FuncOf(variadicIn)
	js.Global().Set("rjs_variadicIn", jsVariadicIn)
	jsVariadicOut := reflectjs.FuncOf(variadicOut)
	js.Global().Set("rjs_variadicOut", jsVariadicOut)
	jsBytesIn := reflectjs.FuncOf(bytesIn)
	js.Global().Set("rjs_bytesIn", jsBytesIn)
	jsBytesOut := reflectjs.FuncOf(bytesOut)
	js.Global().Set("rjs_bytesOut", jsBytesOut)
	select {}
}
