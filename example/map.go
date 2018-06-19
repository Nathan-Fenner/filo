package main

import "fmt"
import "strconv"

// instance for Map::[int string]
func Map_int_string(xs []int, f func(int)string) []string {
  ys := []string{}
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
  
  mapped := Map_int_string(list, func(x int) string {
    return strconv.Itoa(x) + " = " + strconv.Itoa(x)
  })

  fmt.Printf("%+v\n", mapped)
}

