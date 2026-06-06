//go:build !windows && !darwin && !linux

// No app-icon extraction on other platforms; the frontend shows its placeholder.
package services

import "errors"

func fileIconPNG(_ string, _ int) ([]byte, error) {
	return nil, errors.New("unsupported platform")
}
