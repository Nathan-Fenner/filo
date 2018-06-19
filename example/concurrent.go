
package main

import "fmt"

// instance for Req::[int]
type Req_int func() (int, error)

// instance for Concurrently::[int]
func Concurrently_int(actions []Req_int) ([]int, error) {
    out := make([]int, len(actions))
    errs := make(chan error, len(actions))
    answers := make(chan struct{}, len(actions))
    for i := range actions {
        go func(i int) {
            ans, err := actions[i]()
            if err != nil {
                errs <- err
            } else {
                out[i] = ans
                answers <- struct{}{}
            }
        }(i)
    }
    for range out {
        select {
        case <-answers:
        case err := <-errs:
            return nil, err
        }
    }
    return out, nil
}

func main() {
    actions := []Req_int{
        func() (int, error) { return 2, nil },
        func() (int, error) { return 7, nil },
        func() (int, error) { return 3, nil },
        func() (int, error) { return 8, nil },
    }

    out, err := Concurrently_int(actions)
    fmt.Printf("%+v, %+v\n", out, err)
}

