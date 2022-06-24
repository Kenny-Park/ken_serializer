package main

import "fmt"

type Foo struct {
	Kenny  int    `size:"8"`
	Mark   int    `size:"8"`
	Tom    int    `size:"8"`
	James  string `size:"100"`
	Ahreum string `size:"20"`
}

func main() {
	var foo Foo
	var foo2 Foo

	foo = Foo{
		Kenny:  1,
		Mark:   2,
		Tom:    3,
		James:  "james",
		Ahreum: "ahreum",
	}

	fmt.Println(foo)
	// convert to byte array
	b := KenSerializer{}.ToByte(foo)
	fmt.Println(b)

	//convert to struct
	KenSerializer{}.ToStruct(b, &foo2)
	fmt.Println(foo2)
}
