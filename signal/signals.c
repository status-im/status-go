#if defined(IOS_DEPLOYMENT)
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

#elif defined(ANDROID_DEPLOYMENT)
// ======================================================================================
// Android archive compilation using xgo
// ======================================================================================

#include <stddef.h>
#include <stdbool.h>
#include <jni.h>

bool StatusServiceSignalEvent(const char *jsonEvent);

static JavaVM *gJavaVM = NULL;
static jclass JavaClassPtr_StatusService = NULL;
static jmethodID JavaMethodPtr_signalEvent = NULL;

static bool JniLibraryInit(JNIEnv *env);

/*!
 * @brief Get interface to JNI.
 *
 * @return true if thread should be detached from JNI.
 */
static bool JniAttach(JNIEnv **env) {
	jint status;

	if (gJavaVM == NULL) {
		env = NULL;
	}

	status = (*gJavaVM)->GetEnv(gJavaVM, (void **)env, JNI_VERSION_1_6);
	if (status == JNI_EDETACHED) {
		// attach thread to JNI
		//(*gJavaVM)->AttachCurrentThread( gJavaVM, (void **)env, NULL ); // Oracle JNI API
		(*gJavaVM)->AttachCurrentThread(gJavaVM, env, NULL); // Android JNI API
		return true;
	} else if (status != JNI_OK) {
		return false;
	}

	return false;
}

/*!
 * @brief The VM calls JNI_OnLoad when the native library is loaded.
 */
JNIEXPORT jint JNI_OnLoad(JavaVM* vm, void* reserved) {
	bool detach;
	JNIEnv *env;
	int result = JNI_VERSION_1_6;

	gJavaVM = vm;

	// attach thread to JNI
	detach = JniAttach(&env);
	if (env == NULL) {
		// failed
		gJavaVM = NULL;
		return 0;
	}

	if (!JniLibraryInit(env)) {
		// fail loading of JNI library
		result = 0;
	}

	if (detach) {
		// detach thread from JNI
		(*gJavaVM)->DetachCurrentThread(gJavaVM);
	}

	if (result != JNI_VERSION_1_6) {
		gJavaVM = NULL;
	}

	return result;
}

/*!
 * @brief Initialize library.
 */
bool JniLibraryInit(JNIEnv *env) {
	int i;

	JavaClassPtr_StatusService = (*env)->FindClass(env, "im/status/ethereum/module/StatusService");
	if (JavaClassPtr_StatusService == NULL) return false;

	JavaClassPtr_StatusService = (jclass)(*env)->NewGlobalRef(env, JavaClassPtr_StatusService);
	if (JavaClassPtr_StatusService == NULL) return false;

	struct {
        bool bStatic;
        jclass classPtr;
        jmethodID *methodPtr;
        const char *methodId;
        const char *params;
	} javaMethodDescriptors[] = {
		{
		    true,
		    JavaClassPtr_StatusService,
		    &JavaMethodPtr_signalEvent, // &JavaMethodPtr_someNonStaticMethod
            "signalEvent", // someNonStaticMethod
            "(Ljava/lang/String;)V"
        },
	};

	for (i = 0; i < sizeof(javaMethodDescriptors) / sizeof(javaMethodDescriptors[0]); i++) {
		if (javaMethodDescriptors[i].bStatic) {
			*(javaMethodDescriptors[i].methodPtr) = (*env)->GetStaticMethodID(
			    env, javaMethodDescriptors[i].classPtr, javaMethodDescriptors[i].methodId, javaMethodDescriptors[i].params);
		} else {
			*(javaMethodDescriptors[i].methodPtr) = (*env)->GetMethodID(
			    env, javaMethodDescriptors[i].classPtr, javaMethodDescriptors[i].methodId, javaMethodDescriptors[i].params);
		}

		if (*(javaMethodDescriptors[i].methodPtr) == NULL) return false;
	}

	return true;
}

/*!
 * @brief Calls static method signalEvent of class im.status.ethereum.module.StatusService.
 *
 * @param jsonEvent - UTF8 string
 */
bool StatusServiceSignalEvent(const char *jsonEvent) {
	bool detach;
	JNIEnv *env;

	// attach thread to JNI
	detach = JniAttach( &env );
	if (env == NULL) { // failed
		return false;
	}

	jstring javaJsonEvent = NULL;
	if (jsonEvent != NULL) {
		javaJsonEvent = (*env)->NewStringUTF(env, jsonEvent);
	}

	(*env)->CallStaticVoidMethod(env, JavaClassPtr_StatusService, JavaMethodPtr_signalEvent, javaJsonEvent);

	if (javaJsonEvent != NULL) (*env)->DeleteLocalRef(env, javaJsonEvent);

	if (detach) { // detach thread from JNI
		(*gJavaVM)->DetachCurrentThread(gJavaVM);
	}

	return true;
}

#else
// ======================================================================================
// cgo compilation (for desktop platforms and local tests)
// ======================================================================================

#include <stdio.h>
#include <stddef.h>
#include <stdbool.h>
#include "_cgo_export.h"

typedef void (*callback)(const char *jsonEvent);
callback gCallback = 0;

bool StatusServiceSignalEvent(const char *jsonEvent) {
	if (gCallback) {
		gCallback(jsonEvent);
	} else {
		NotifyNode((char *)jsonEvent); // re-send notification back to status node
	}

	return true;
}

void SetEventCallback(void *cb) {
	gCallback = (callback)cb;
}

#endif
