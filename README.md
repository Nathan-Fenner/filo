# Filo
A preprocessor that adds generics (in the form of templating) to Go.

```go
package main

import "fmt"
import "strconv"

func Map::[#A, #B](xs []#A, f func(#A)#B) []#B {
  ys := []#B{}
  for _, x := range xs {
    ys = append(ys, f(x))
  }
  return ys
}

func transform(x int) string {
  return fmt.Sprintf("!%d!", x)
}

func main() {
  list := []int{1, 2, 3, 4}
  
  mapped := Map::[int, string](list, func(x int) string {
    return strconv.Itoa(x) + " = " + strconv.Itoa(x)
  })

  fmt.Printf("%+v\n", mapped)
}
```

Then run `go run filo.go gen example/map.filo` to create the file `example/map.go`. Then run it how you normally would.

# Features

Top-level structs, interfaces, and functions can be generic. For example,

```go
type Producer::[#T] interface {
  Produce() #T
}

type Maker::[#X] func()#X

func (m Maker::[#X]) Produce() #X {
  return m()
}
```

# Limitations

Receiver functions (methods) cannot be generic (*but* generic types can still have receiver functions - see the above).
This drastically simplifies Go's runtime semantics, since it means the receiver function set for every type is finite, fixed, and obvious.

Name collisions are possible between generated terms; this can only occur if your type names contain underscores.

Filo itself does relatively little correctness checking. In particular, templates aren't type-checked (or even parsed for syntax) until they're expanded.

Filo only understands single-file projects at the moment. Since it just acts as a preprocessor, this isn't actually so bad, as long as you're okay with running the tool separately on every Filo file in your project, whenever they change.

Filo doesn't even lex the source file it's given, so it will attempt to modify comments or strings that look like they contain generic code.

Filo doesn't infer any type parameters. This makes it much easier to implement the templating system, but harder to use it. Since the preprocessor deliberately avoids understanding Go code, this will be fairly hard to fix.

