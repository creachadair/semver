// Copyright (C) 2024 Michael J. Fromberger. All Rights Reserved.

package semver_test

import (
	"fmt"
	"log"

	"github.com/creachadair/semver"
)

func Example() {
	v, err := semver.Parse("1.5.3-rc1.4+modified")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("version:", v)
	fmt.Println("core:", v.Core())
	fmt.Println("pre:", v.PreRelease())
	fmt.Println("build:", v.Build())
	fmt.Println("clean:", v.WithBuild(""))

	w := semver.New(1, 5, 3).WithPreRelease("rc1.4")
	fmt.Println("equiv:", v, w, v.Equiv(w))

	// Output:
	// version: 1.5.3-rc1.4+modified
	// core: 1.5.3
	// pre: rc1.4
	// build: modified
	// clean: 1.5.3-rc1.4
	// equiv: 1.5.3-rc1.4+modified 1.5.3-rc1.4 true
}

func ExampleClean() {
	const dirty = " v1.2-rc3..1\t"

	fmt.Printf("dirty: %q\n", dirty)
	fmt.Println("clean:", semver.Clean(dirty))
	// Output:
	// dirty: " v1.2-rc3..1\t"
	// clean: 1.2.0-rc3.1
}
