package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"net/http"
	"runtime"

	"github.com/gopherjs/gopherjs/compiler"
	"github.com/gopherjs/gopherjs/js"
)

// newImportContext creates an ImportContext instance, which downloads
// precompiled package archives.
func newImportContext() *compiler.ImportContext {
	archives := make(map[string]*compiler.Archive)
	packages := make(map[string]*types.Package)
	importContext := &compiler.ImportContext{
		Packages: packages,
		Import: func(path string) (*compiler.Archive, error) {
			if pkg, found := archives[path]; found {
				return pkg, nil
			}

			var respData []byte
			var err error

			// Precompiled archives are located at "pkg/<import path>.a.js" relative
			// URL, convert that to the absolute URL http.Get() needs.
			url := "https://cdn.jsdelivr.net/gh/naorzr/gopherjs-runtime@1.18.0-beta3/pkg/" + path + ".a.js"
			resp, err := http.Get(url)
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()

			respData, err = io.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}

			pkg, err := compiler.ReadArchive(path+".a", bytes.NewReader(respData))
			if err != nil {
				return nil, err
			}
			archives[path] = pkg

			if err := pkg.RegisterTypes(packages); err != nil {
				return nil, err
			}

			return pkg, nil
		},
	}
	return importContext
}

// Playground implements Go code compilation and execution within a web browser
// context.
type Playground struct {
	importContext *compiler.ImportContext
}

// Compile the given Go source code.
//
// Returns generated JS code that can be evaluated, or an error if compilation
// fails.
func (p *Playground) Compile(code string, callback func(error, string)) {
	go func() {
		fileSet := token.NewFileSet()

		file, err := parser.ParseFile(fileSet, "prog.go", []byte(code), parser.ParseComments)
		if err != nil {
			callback(err, "")
		}

		mainPkg, err := compiler.Compile("main", []*ast.File{file}, fileSet, p.importContext, false)
		if err != nil {
			callback(err, "")
		}

		allPkgs, _ := compiler.ImportDependencies(mainPkg, p.importContext.Import)

		jsCode := bytes.NewBuffer(nil)
		compiler.WriteProgramCode(allPkgs, &compiler.SourceMapFilter{Writer: jsCode}, runtime.Version())
		callback(nil, jsCode.String())
	}()
}

// Run the compiled JS code.
//
// If execution throws an exception, it will be caught and returned as an error.
func (p *Playground) Run(compiled string) (returnedError error) {
	defer func() {
		// JS errors are converted into Go panics, so that we can recover from them.
		e := recover()
		if e == nil {
			return
		}

		// We got a JS error, propagate it as-is.
		if err, ok := e.(*js.Error); ok {
			returnedError = err
		}

		// Some other unknown kind of panic, wrap it in an error.
		returnedError = fmt.Errorf("compiled code paniced: %v", e)
	}()

	js.Global.Call("eval", compiled)
	return nil
}

func main() {
	// Create a playground object. It will exist for the lifetime of the
	// application, allowing it to cache precompiled packages between runs.
	// This improves performance by avoiding unnecessary recompilation.
	p := Playground{
		importContext: newImportContext(),
	}

	js.Global.Set("__GOPHERJS__", js.MakeWrapper(&p))
}
