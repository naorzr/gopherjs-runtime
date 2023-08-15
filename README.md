# In browser Go compiler

This package provides a way to run Go code in the browser without needing a server, using [GopherJS](https://github.com/gopherjs/gopherjs).   

heavily inspired by [Gopherjs Playground](https://gopherjs.github.io/playground/)

## Features

- **Compile Go Code**: Transpile Go code into JavaScript that can be run in the browser.
- **Non-blocking Compilation**: Utilize callback functions to handle the compiled code or any errors.
- **Precompiled Package Support**: Downloads and caches precompiled package archives from a specified URL.
- **Exception Handling**: Catches exceptions thrown during execution and returns them as errors.

## Usage

### Including the Gopherjs Compiler


```html
<script src="https://cdn.jsdelivr.net/gh/naorzr/gopherjs-runtime@1.18.0-beta3/main.js"></script>
```

### Compiling Go Code

You can then use the global `__GOPHERJS__` object to compile Go code:

```javascript
__GOPHERJS__.Compile(`
package main

import "fmt"

func main() {
  fmt.Println("Hello, playground")
}
`, function(result, err) {
  if (err) {
    console.error("Compilation error:", err);
    return;
  }

  // Handle the compiled code (result)
});
```

### Running Compiled Code

The compile code is a string of JavaScript code that can be executed in the browser.

You can use `eval`, `new Function`, or any other method to execute the code:

conviniently, the `__GOPHERJS__` object exposes a `Run` method that can be used to run the compiled code:

```javascript
__GOPHERJS__.Run(compiledCode);
```

## Importing Precompiled Packages

The logic includes the ability to download and cache precompiled package archives.   
By default, the URL is set to `"https://cdn.jsdelivr.net/gh/naorzr/gopherjs-runtime@1.18.0-beta3/playground/pkg/"`.   
Make sure the version tag matches the version used in your `go.mod` file
## How Does It Work

Gopherjs is a library written in Go that enables compiling Go code into a single executable javascript file.
currently, the Gopherjs compiler is available only on the server side, since it's written in Go, but using the Gopherjs compiler, we can compile the compiler to a javascript file, that will enable us to use it on the client side.

the `main.go` file describes the process of using the compiler and exposing globals, and then that file is compiled to main.js using the Gopherjs compiler.


## Development

To update the entire playground environment, just run `https://github.com/naorzr/gopherjs-runtime`. It will install your local version of GopherJS compiler, build the playground, make a temporary copy of Go to /tmp/gopherjsplayground_goroot, rebuild and copy the standard library into the `pkg` directory.

Working on the playground application itself is made easier by using the `gopherjs serve` command to rebuild and serve the sample playground every time you refresh the browser.

```bash
gopherjs serve
```

Then open <http://localhost:8080/github.com/gopherjs/gopherjs.github.io/playground>.

## Upgrading GopherJS release

Step 1: Update the version in `go.mod` to the latest version.

```shell
VERSION="$(go list -m -versions -f "{{ range .Versions }}{{ println . }}{{ end }}" github.com/gopherjs/gopherjs | tail -n 1)"
echo "$VERSION"
go get -v "github.com/gopherjs/gopherjs@$VERSION"
go mod tidy
```

Step 2: Update the jsdelivr URL in `main.go` to the corresponding version.

Step 3: rebuild `main.go` to `main.js` using the GopherJS compiler.


## Contributions

Please open an issue on GitHub for support or contributions.


## License

MIT License (c)