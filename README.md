# reflectjs
This library replaces the builtin syscall/js.FuncOf in go. Instead of passing in a `func(js.value, []js.value) any`, you may pass in any function. The arguments will be converted from js.Value to the input argument automatically using reflection. The return values will also be reflected back into javascript values.

## example
```go
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

func main() {
    jsDog := reflectjs.FuncOf(dog)
    js.Global().Set("rjs_dog", jsDog)
}
```
Calling rjs_dog from javascript would then look something like this:
```js
rjs_dog({
    "bar": () => 1,
    "Baz": 1.23,
    "Dog": {
        "Age": 12,
        "Name": "Charlie"
    }
})
> 2023/02/17 18:43:15 main.Foo{Bar:(func() int)(0x14db0000), Baz:1.23, Dog:main.Dog{Age:12, Name:"Charlie"}}
> {Age: 12, Name: 'Charlie'}
```

## returning functions / accepting functions as arguments
You may accept functions as arguments or return functions and everything will work as expected. The only caveat is that returning functions causes a slight memory leak (because there is currently no way to know when the function should be released), so avoid it where possible.

I'll probably fix that using the finalization registry eventually.