#include <stddef.h>
#include <stdbool.h>
#include <jni.h>


bool GethServiceSignalEvent( const char *jsonEvent );

static JavaVM *gJavaVM = NULL;
static jclass JavaClassPtr_GethService = NULL;
static jmethodID JavaMethodPtr_signalEvent = NULL;

static bool JniLibraryInit( JNIEnv *env );

/*!
 * @brief Get interface to JNI.
 * 
 * @return true if thread should be detached from JNI.
 */
static bool JniAttach( JNIEnv **env )
{
	jint status;
	
	if (gJavaVM == NULL)
	{
		env = NULL;
	}
	
	status = (*gJavaVM)->GetEnv( gJavaVM, (void **)env, JNI_VERSION_1_6);
	if (status == JNI_EDETACHED)
	{
		// attach thread to JNI
		//(*gJavaVM)->AttachCurrentThread( gJavaVM, (void **)env, NULL );		// Oracle JNI API
		(*gJavaVM)->AttachCurrentThread( gJavaVM, env, NULL );			// Android JNI API
		return true;
	}
	else if (status != JNI_OK)
	{
		return false;
	}
	
	return false;
}

/*!
 * @brief The VM calls JNI_OnLoad when the native library is loaded.
 */
JNIEXPORT jint JNI_OnLoad(JavaVM* vm, void* reserved)
{
	bool detach;
	JNIEnv *env;
	int result = JNI_VERSION_1_6;

	gJavaVM = vm;

	// attach thread to JNI
	detach = JniAttach( &env );
	if (env == NULL)
	{
		// failed
		gJavaVM = NULL;
		return 0;
	}
	
	if (!JniLibraryInit( env ))
	{
		// fail loading of JNI library
		result = 0;
	}

	if (detach)
	{
		// detach thread from JNI
		(*gJavaVM)->DetachCurrentThread( gJavaVM );
	}
	
	if (result != JNI_VERSION_1_6)
	{
		gJavaVM = NULL;
	}
	
	return result;
}

/*!
 * @brief Initialize library.
 */
bool JniLibraryInit( JNIEnv *env )
{
	int i;
	
	JavaClassPtr_GethService = (*env)->FindClass( env, "com/statusim/geth/service/GethService" );
	if (JavaClassPtr_GethService == NULL) return false;
	JavaClassPtr_GethService = (jclass)(*env)->NewGlobalRef( env, JavaClassPtr_GethService );
	if (JavaClassPtr_GethService == NULL) return false;
	
	struct { bool bStatic; jclass classPtr; jmethodID *methodPtr; const char *methodId; const char *params; } javaMethodDescriptors[] =
	{
		{ true, JavaClassPtr_GethService, &JavaMethodPtr_signalEvent, 								"signalEvent", 							"(Ljava/lang/String;)V" },
			
//		{ false, JavaClassPtr_GethService, 	&JavaMethodPtr_someNonStaticMethod, 						"someNonStaticMethod", 					"(Ljava/lang/String;)V" },
	};

	for (i = 0; i < sizeof(javaMethodDescriptors) / sizeof(javaMethodDescriptors[0]); i++)
	{
		if (javaMethodDescriptors[i].bStatic)
		{
			*(javaMethodDescriptors[i].methodPtr) = (*env)->GetStaticMethodID( env, javaMethodDescriptors[i].classPtr, javaMethodDescriptors[i].methodId, javaMethodDescriptors[i].params );
		}
		else
		{
			*(javaMethodDescriptors[i].methodPtr) = (*env)->GetMethodID( env, javaMethodDescriptors[i].classPtr, javaMethodDescriptors[i].methodId, javaMethodDescriptors[i].params );
		}

		if (*(javaMethodDescriptors[i].methodPtr) == NULL) return false;
	}
	
	return true;
}

/*!
 * @brief Calls static method signalEvent of class com.statusim.GethService.
 * 
 * @param jsonEvent - UTF8 string
 */
bool GethServiceSignalEvent( const char *jsonEvent )
{
	bool detach;
	JNIEnv *env;

	// attach thread to JNI
	detach = JniAttach( &env );
	if (env == NULL)
	{
		// failed
		return false;
	}
	
	jstring javaJsonEvent = NULL;
	if (jsonEvent != NULL)
	{
		javaJsonEvent = (*env)->NewStringUTF( env, jsonEvent );
	}

	(*env)->CallStaticVoidMethod( env, JavaClassPtr_GethService, JavaMethodPtr_signalEvent, javaJsonEvent );

	if (javaJsonEvent != NULL) (*env)->DeleteLocalRef( env, javaJsonEvent );

	if (detach)
	{
		// detach thread from JNI
		(*gJavaVM)->DetachCurrentThread( gJavaVM );
	}
	
	return true;
}
