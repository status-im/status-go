FROM statusteam/xgo:1.9.2

ADD update_ios.sh /update_ios.sh
ENV UPDATE_IOS /update_ios.sh
RUN chmod +x $UPDATE_IOS

RUN \
  IOS_SDK_PATH=https://s3.amazonaws.com/farazdagi/status-im/iPhoneSimulator9.3.sdk.tar.gz && \
  $FETCH $IOS_SDK_PATH 460423bf776e651a84c6e1bc035dbce23f93e685 && \
  $UPDATE_IOS /iPhoneSimulator9.3.sdk.tar.gz && \
  rm -rf /iPhoneSimulator9.3.sdk.tar.gz
