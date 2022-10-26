// This program frees allocated memory twice.
//
// Compile to Go: `$ ccgo -o main.go doublefree.c`.
//
// To run the resulting Go code: `$ go run main.go`.
//
// To run the resulting Go code with memgrind: `$ go run -tags=libc.memgrind main.go`.

#include <stdlib.h>

int main() {
	void *p = malloc(42);
	free(p);
	free(p);
}

