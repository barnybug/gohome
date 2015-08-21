package gogsmmodem

import "fmt"

func ExampleParseTime() {
	t := parseTime("14/02/01,15:07:43+00")
	fmt.Println(t)
	// Output:
	// 2014-02-01 15:07:43 +0000 UTC
}

func ExampleStartsWith() {
	fmt.Println(startsWith("abc", "ab"))
	fmt.Println(startsWith("abc", "b"))
	// Output:
	// true
	// false
}

func ExampleQuotes() {
	args := []interface{}{"a", 1, "b"}
	fmt.Println(quotes(args))
	// Output:
	// "a",1,"b"
}

func ExampleUnquotes() {
	fmt.Println(unquotes(`"a,comma",1,"b"`))
	// Output:
	// [a,comma 1 b]
}

func ExampleGsmEncode() {
	fmt.Printf("%q\n", gsmEncode("abcdefghijklmnopqrstuvwxyz"))
	fmt.Printf("%q\n", gsmEncode("ABCDEFGHIJKLMNOPQRSTUVWXYZ"))
	fmt.Printf("%q\n", gsmEncode("0123456789"))
	fmt.Printf("%q\n", gsmEncode(".,+-*/ "))
	fmt.Printf("%q\n", gsmEncode("°"))
	fmt.Printf("%q\n", gsmEncode("@£"))
	fmt.Printf("%q\n", gsmEncode("{}"))
	// Output:
	// "abcdefghijklmnopqrstuvwxyz"
	// "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	// "0123456789"
	// ".,+-*/ "
	// ""
	// "\x00\x01"
	// "\x1b(\x1b)"
}
