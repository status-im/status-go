#include <stdio.h>
#include "_cgo_export.h"

int doNewAccount() {
	char *account = NewAccount("badpassword", "/home/dwhitena/.ethereum/keystore");
	printf("%s\n", account);
	return 0;
} 
