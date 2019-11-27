// +build nimbus

package nimbusbridge

/*

#include <libnimbus.h>

// onMessageHandler gateway function
void onMessageHandler_cgo(received_message * msg, void* udata)
{
	void onMessageHandler(received_message* msg, void* udata);
	onMessageHandler(msg, udata);
}
*/
import "C"
