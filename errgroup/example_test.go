package errgroup_test

import (
	"fmt"

	"github.com/wood-jp/xerrors/errclass"
	"github.com/wood-jp/xerrors/errgroup"
)

func ExampleGroup_Go() {
	g := errgroup.New()
	g.Go(func() error {
		panic("something went wrong")
	})
	err := g.Wait()
	fmt.Println(errclass.GetClass(err))
	// Output:
	// panic
}
