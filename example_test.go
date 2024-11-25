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
	fmt.Println("release:", v.Release())
	fmt.Println("build:", v.Build())
	fmt.Println("clean:", v.WithBuild(""))

	w := semver.New(1, 5, 3).WithRelease("rc1.4")
	fmt.Println("equiv:", v, w, v.Equiv(w))

	// Output:
	// version: 1.5.3-rc1.4+modified
	// core: 1.5.3
	// release: rc1.4
	// build: modified
	// clean: 1.5.3-rc1.4
	// equiv: 1.5.3-rc1.4+modified 1.5.3-rc1.4 true
}

func ExampleParse() {
	v, err := semver.Parse(semver.Clean(" v1.2-alpha..9.\n"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(v)
	// Output:
	// 1.2.0-alpha.9
}

func ExampleClean() {
	const dirty = " v1.2-rc3..1\t"

	fmt.Printf("dirty: %q\n", dirty)
	fmt.Println("clean:", semver.Clean(dirty))
	// Output:
	// dirty: " v1.2-rc3..1\t"
	// clean: 1.2.0-rc3.1
}

func ExampleV_WithCore() {
	v := semver.MustParse("1.1.3+unstable")
	w := v.WithCore(2, 0, -1)

	fmt.Println("v:", v)
	fmt.Println("w:", w)
	// Output:
	// v: 1.1.3+unstable
	// w: 2.0.3+unstable
}

func ExampleV_Add() {
	v := semver.New(1, 5, 3)
	w := v.Add(0, -10, 2)

	fmt.Println("v:", v)
	fmt.Println("w:", w)
	// Output:
	// v: 1.5.3
	// w: 1.0.5
}
