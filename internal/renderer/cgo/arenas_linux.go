//go:build cgo && !nocgo

package cgo

// glibc gives every thread that renders an arena that keeps what it frees. The cap
// has to be set in a constructor, before the Go runtime starts its threads.

/*
#include <malloc.h>
#include <stdlib.h>

__attribute__((constructor)) static void pwikit_limit_arenas(void) {
#ifdef M_ARENA_MAX
	if (getenv("MALLOC_ARENA_MAX") == NULL) {
		mallopt(M_ARENA_MAX, 2);
	}
#endif
}
*/
import "C"
