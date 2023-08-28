package proxy

/*
#include <stdio.h>

void goCallback_cgo(char * json, int cbType) {
	printf("inside goCallback_cgo\n");
	void goCallback(char *, int);
	goCallback(json, cbType);
}
*/
import "C"
