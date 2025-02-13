// This program leaks memory.
//
// Compile to Go: `$ ccgo -o main.go leak.c`.
//
// To run the resulting Go code: `$ go run main.go`.
//
// To run the resulting Go code with memgrind: `$ go run -tags=libc.memgrind main.go`.

#include <stdlib.h>

int main() {
	malloc(42);
}
