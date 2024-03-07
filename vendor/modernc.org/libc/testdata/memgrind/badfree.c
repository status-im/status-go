// This program attempts to free a pointer not acquired by malloc/calloc/realloc.
//
// Compile to Go: `$ ccgo -o main.go badfree.c`.
//
// To run the resulting Go code: `$ go run main.go`.
//
// To run the resulting Go code with memgrind: `$ go run -tags=libc.memgrind main.go`.

#include <stdlib.h>

int main() {
	int i;
	free(&i);
}

