package automata

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

func BasicLitToValue(t *ast.BasicLit) reflect.Value {
	var i interface{}
	switch t.Kind {
	case token.STRING:
		i = strings.Trim(t.Value, "\"")
	case token.INT:
		i, _ = strconv.ParseInt(t.Value, 10, 32)
	case token.FLOAT:
		i, _ = strconv.ParseFloat(t.Value, 64)
	}
	return reflect.ValueOf(i)
}

func DynamicCall(obj interface{}, call string) (err error) {
	as, _ := parser.ParseExpr(call)
	ce, ok := as.(*ast.CallExpr)
	if !ok {
		return errors.New("Didn't parse to CallExpr")
	}

	instance := reflect.ValueOf(obj)
	fname := fmt.Sprint(ce.Fun)
	method := instance.MethodByName(fname)
	if method.IsValid() {
		var args []reflect.Value
		for _, expr := range ce.Args {
			var v reflect.Value
			switch t := expr.(type) {
			case *ast.BasicLit:
				v = BasicLitToValue(t)
			case *ast.Ident:
				switch t.Name {
				case "true":
					v = reflect.ValueOf(true)
				case "false":
					v = reflect.ValueOf(false)
				}
			default:
				return errors.New(fmt.Sprintf("Expression: %v not understood", t))
			}
			args = append(args, v)
		}

		defer func() {
			if r := recover(); r != nil {
				err = errors.New(fmt.Sprintf("Error calling: %s %s", call, r))
			}
		}()
		method.Call(args)
	} else {
		err = errors.New(fmt.Sprintf("Error: %s not found", call))
	}
	return
}

var reSub = regexp.MustCompile(`\$(\w+)`)

func Substitute(s string, vals map[string]string) string {
	return reSub.ReplaceAllStringFunc(s, func(k string) string {
		if v, ok := vals[k[1:]]; ok {
			return v
		}
		return k
	})
}
