rm -rf ../../web/connectreactive/ios/ReactNativeModules/Common.framework
gomobile bind -target=ios .
mv ./Common.framework ../../web/connectreactive/ios/ReactNativeModules/Common.framework

gomobile bind -target=android .
mv ./common.aar ../../web/connectreactive/android/app/libs/common.aar