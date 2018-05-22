/*
	Package benchmarks contains tests that can be used
	to benchmark cluster components.


	Example usage:

		1. Start a Whisper node with mail server capability:
			./build/bin/statusd \
				-networkid=4 \
				-maxpeers=100 \
				-shh \
				-shh.pow=0.002 \
				-shh.mailserver \
				-shh.passwordfile=./static/keys/wnodepassword \
				-log DEBUG
		2. Generate some messages:
			go test -v -timeout=30s -run TestSendMessages ./t/benchmarks \
				-peerurl=$ENODE_ADDR \
				-msgcount=200 \
				-msgbatchsize=50
		3. Retrieve them from mail server:
			go test -v -timeout=30s -run TestConcurrentMailserverPeers ./t/benchmarks \
				-peerurl=$ENODE_ADDR \
				-msgcount=200
*/

package benchmarks
