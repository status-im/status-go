FROM statusteam/xgo:1.10.x

# ios installer doesn't work with the default tar binary in Docker
# (divan: I stole this solution here: https://github.com/coreos/bugs/issues/1095#issuecomment-350574389)
# https://github.com/moby/moby/issues/19647
RUN apt-get update \
	&& apt-get install -y --no-install-recommends bsdtar \
	&& apt-get clean \
	&& cp $(which tar) $(which tar)~ \
    && ln -sf $(which bsdtar) $(which tar)

ADD update_ios.sh /update_ios.sh
ENV UPDATE_IOS /update_ios.sh
RUN chmod +x $UPDATE_IOS

RUN \
  IOS_SDK_PATH=https://s3.amazonaws.com/farazdagi/status-im/iPhoneSimulator9.3.sdk.tar.gz && \
  $FETCH $IOS_SDK_PATH 460423bf776e651a84c6e1bc035dbce23f93e685 && \
  $UPDATE_IOS /iPhoneSimulator9.3.sdk.tar.gz && \
  rm -rf /iPhoneSimulator9.3.sdk.tar.gz
