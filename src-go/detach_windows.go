//go:build windows

package main

import "errors"

func detach() (int, error) {
	return 0, errors.New("-detach is not supported on windows")
}
