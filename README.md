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
