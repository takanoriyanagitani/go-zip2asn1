package util

import (
	mi "github.com/takanoriyanagitani/go-zip2asn1"
)

func ComposeErr[T, U, V any](
	f func(T) (U, error),
	g func(U) (V, error),
) func(T) (V, error) {
	return mi.ComposeErr(f, g)
}
