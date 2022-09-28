rm -rf /Users/aleksandrgorbunov/srv/connectclub-reactnative/ios/ReactNativeModules/Common.framework
gomobile bind -target=ios .
mv ./Common.framework /Users/aleksandrgorbunov/srv/connectclub-reactnative/ios/ReactNativeModules/Common.framework

gomobile bind -target=android .
mv ./common.aar /Users/aleksandrgorbunov/srv/connectclub-reactnative/android/app/libs/common.aar