# Android Kotlin Example


## Requirements
- Android Studio


## Running this example
These instructions should be executed in the terminal: 
```bash
# Clone the repository
git clone https://github.com/waku-org/go-waku.git
cd go-waku

# Set required env variables
export ANDROID_NDK_HOME=/path/to/android/ndk
export ANDROID_HOME=/path/to/android/sdk/

# Build the .jar
make mobile-android

# Copy the jar into `libs/` folder
cp ./build/lib/gowaku.aar ./examples/android-kotlin/app/libs/.
```

Open the project in Android Studio and run the example app.


## Help wanted!
- Is it possible to build go-waku automatically by executing `make mobile-android` and copying the .jar automatically into `libs/` in Android Studio?
- Permissions should be requested on runtime
- Determine the required permission to fix this:
```
2022-04-07 19:29:27.542 20042-20068/com.example.waku E/GoLog: 2022-04-07T23:29:27.542Z	ERROR	basichost	basic/basic_host.go:327	failed to resolve local interface addresses	{"error": "route ip+net: netlinkrib: permission denied"}
```
- The example app blocks the main thread and code in general could be improved
