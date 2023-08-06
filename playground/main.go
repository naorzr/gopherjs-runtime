package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"go/types"
	"io/ioutil"
	"net/http"
	"runtime"
	"strings"

	"github.com/gopherjs/gopherjs/compiler"
	"github.com/gopherjs/gopherjs/js"
)

type Line map[string]string

var output []Line
var fileSet *token.FileSet
var importContext *compiler.ImportContext
var packages map[string]*compiler.Archive

func main() {
	packages = make(map[string]*compiler.Archive)
	importContext = &compiler.ImportContext{
		Packages: make(map[string]*types.Package),
		Import: func(path string) (*compiler.Archive, error) {
			if pkg, found := packages[path]; found {
				return pkg, nil
			}

			var respData []byte
			var err error

			resp, err := http.Get("https://cdn.jsdelivr.net/gh/naorzr/gopherjs-runtime@master/playground/pkg/" + path + ".a.js")
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()

			respData, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}

			if err != nil {
				return nil, err
			}

			pkg, err := compiler.ReadArchive(path+".a", bytes.NewReader(respData))
			if err != nil {
				// Handle error
				return nil, err
			}
			packages[path] = pkg

			if err := pkg.RegisterTypes(importContext.Packages); err != nil {
				js.Global.Call("eval", js.InternalObject("console.error('error registering types package: "+path+"')"))
				return nil, err
			}

			return pkg, nil
		},
	}
	fileSet = token.NewFileSet()
	setupEnvironment()
	js.Global.Set("runCode", js.MakeFunc(runCode))
	js.Global.Set("runCodeAsync", js.MakeFunc(runCodeAsync))
}

func runCodeAsync(this *js.Object, arguments []*js.Object) interface{} {
	callback := arguments[1] // Assuming you pass a JS callback as the second argument

	go func() {
		result := runCode(this, arguments[:1]) // You can adjust the arguments slice as needed

		// Once we have the result, we call the JS callback with the result
		callback.Invoke(result)
	}()

	return nil
}

func runCode(this *js.Object, arguments []*js.Object) interface{} {
	code := arguments[0].String()
	escaped, err := json.Marshal(code)
	if err != nil {
		fmt.Println("Error during marshalling:", err)
		return nil
	}

	// log code
	js.Global.Call("eval", js.InternalObject("console.log('code: "+string(escaped)+"')"))

	file, err := parser.ParseFile(fileSet, "prog.go", []byte(code), parser.ParseComments)
	if err != nil {
		if list, ok := err.(scanner.ErrorList); ok {
			var outputErr []Line
			for _, entry := range list {
				outputErr = append(outputErr, Line{"type": "err", "content": entry.Error()})
			}
			js.Global.Get("console").Call("error", "error parsing package: main")
			return outputErr
		}
		js.Global.Get("console").Call("error", "error parsing package2: main")
		return err.Error()
	}

	mainPkg, err := compiler.Compile("main", []*ast.File{file}, fileSet, importContext, false)
	if err != nil {
		if list, ok := err.(compiler.ErrorList); ok {
			var outputErr []Line
			for _, entry := range list {
				outputErr = append(outputErr, Line{"type": "err", "content": entry.Error()})
			}
			js.Global.Get("console").Call("error", "error compiling package: "+mainPkg.Name)

			return outputErr
		}
		js.Global.Get("console").Call("error", "error compiling package2: "+mainPkg.Name)
		return err.Error()
	}

	allPkgs, _ := compiler.ImportDependencies(mainPkg, importContext.Import)

	jsCode := bytes.NewBuffer(nil)
	jsCode.WriteString("try{\n")
	compiler.WriteProgramCode(allPkgs, &compiler.SourceMapFilter{Writer: jsCode}, runtime.Version())
	jsCode.WriteString("} catch (err) {\nconsole.error(err.message);\n}\n")
	js.Global.Set("$checkForDeadlock", true)
	js.Global.Call("eval", js.InternalObject(jsCode.String()))

	return jsCode.String()
}

func setupEnvironment() {
	js.Global.Set("goPrintToConsole", js.InternalObject(func(b []byte) {
		lines := strings.Split(string(b), "\n")
		if len(output) == 0 || output[len(output)-1]["type"] != "out" {
			output = append(output, Line{"type": "out", "content": ""})
		}
		output[len(output)-1]["content"] += lines[0]
		for i := 1; i < len(lines); i++ {
			output = append(output, Line{"type": "out", "content": lines[i]})
		}
	}))
	js.Global.Set("goPanicHandler", js.InternalObject(func(msg string) {
		output = append(output, Line{"type": "err", "content": "panic: " + msg})
	}))
}
