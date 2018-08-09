// +build library,darwin

// ======================================================================================
// iOS framework compilation using xgo
// ======================================================================================

#include <stddef.h>
#include <stdbool.h>
#include <stdlib.h>
#include <objc/objc.h>
#include <objc/runtime.h>
#include <objc/message.h>

static id statusServiceClassRef = nil;
static SEL statusServiceSelector = nil;

static bool initLibrary() {
    if (statusServiceClassRef == nil) {
        statusServiceClassRef = objc_getClass("Status");
        if (statusServiceClassRef == nil) return false;
    }

    if (statusServiceSelector == nil) {
        statusServiceSelector = sel_getUid("signalEvent:");
        if (statusServiceSelector == nil) return false;
    }

    return true;
}


/*!
 * @brief Calls static method signalEvent of class GethService.
 *
 * @param jsonEvent - UTF8 string
 *
 * @note Definition of signalEvent method.
 *  + (void)signalEvent:(const char *)json
 */
bool StatusServiceSignalEvent(const char *jsonEvent) {
    if (!initLibrary()) return false;

    void (*action)(id, SEL, const char *) = (void (*)(id, SEL, const char *)) objc_msgSend;
    action(statusServiceClassRef, statusServiceSelector, jsonEvent);

    return true;
}

void SetEventCallback(void *cb) {
}

