//go:build windows

package main

// relaxUmask has no Windows counterpart: file access there is governed by
// inherited ACLs, not a umask.
func relaxUmask() {}
