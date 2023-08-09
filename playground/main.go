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

func consoleErrorf(format string, args ...interface{}) {
	js.Global.Get("console").Call("error", fmt.Sprintf(format, args...))
}

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
			url := "https://cdn.jsdelivr.net/gh/naorzr/gopherjs-runtime@master/playground/pkg/" + path + ".a.js"
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
func (p *Playground) Compile(code string) (string, error) {
	fileSet := token.NewFileSet()

	file, err := parser.ParseFile(fileSet, "prog.go", []byte(code), parser.ParseComments)
	if err != nil {
		return "", err
	}

	mainPkg, err := compiler.Compile("main", []*ast.File{file}, fileSet, p.importContext, false)
	if err != nil {
		return "", err
	}

	allPkgs, _ := compiler.ImportDependencies(mainPkg, p.importContext.Import)

	jsCode := bytes.NewBuffer(nil)
	compiler.WriteProgramCode(allPkgs, &compiler.SourceMapFilter{Writer: jsCode}, runtime.Version())
	return jsCode.String(), nil
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
	// Create a playground object. Its lifetime will be equal to the lifetime of
	// the event handler on the button, which will allow it to cache precompiled
	// packages between the runs.
	p := Playground{
		importContext: newImportContext(),
	}

	// Obtain references to the relevant DOM nodes.
	codeEl := js.Global.Get("document").Call("getElementById", "code")
	if codeEl == nil {
		panic("Can't find code input field.")
	}
	runEl := js.Global.Get("document").Call("getElementById", "run")
	if runEl == nil {
		panic("Can't find 'Run' button.")
	}

	// Install a click handler to the button. Instead of polluting global
	// namespace, we just register an anonymous function.
	runEl.Call("addEventListener", "click", func(event *js.Object) {
		// Because compilation may involve blocking calls (e.g. network requests),
		// it must be done in a goroutine to avoid blocking event loop. We kick off
		// the goroutine and the event handler completed immediately.
		go func() {
			code := codeEl.Get("value").String()
			println("Compiling the code:\n", code)
			compiled, err := p.Compile(code)
			if err != nil {
				consoleErrorf("Failed to compile the code:\n%s", err)
			}
			err = p.Run(compiled)
			if err != nil {
				consoleErrorf("Failed to execute the code:\n%s", err)
			}
			println("Execution completed without errors.")
		}()
	})
}
