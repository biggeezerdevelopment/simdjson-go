//go:build amd64

package scanner

import (
	"golang.org/x/sys/cpu"
)

func hasAVX2() bool {
	return cpu.X86.HasAVX2
}

func hasSSE42() bool {
	return cpu.X86.HasSSE42
}