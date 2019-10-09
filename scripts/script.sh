#!/bin/bash

echo
echo " ____    _____      _      ____    _____ "
echo "/ ___|  |_   _|    / \    |  _ \  |_   _|"
echo "\___ \    | |     / _ \   | |_) |   | |  "
echo " ___) |   | |    / ___ \  |  _ <    | |  "
echo "|____/    |_|   /_/   \_\ |_| \_\   |_|  "
echo
echo "Build your first network (BYFN) end-to-end test"
echo
CHANNEL_NAME="$1"
DELAY="$2"
LANGUAGE="$3"
TIMEOUT="$4"
VERBOSE="$5"
: ${CHANNEL_NAME:="mychannel1"}
: ${DELAY:="3"}
: ${LANGUAGE:="golang"}
: ${TIMEOUT:="10"}
: ${VERBOSE:="false"}
LANGUAGE=`echo "$LANGUAGE" | tr [:upper:] [:lower:]`
COUNTER=1
MAX_RETRY=10

CC_SRC_PATH="github.com/chaincode/src/token/"
if [ "$LANGUAGE" = "node" ]; then
	CC_SRC_PATH="/opt/gopath/src/github.com/chaincode/chaincode_example02/node/"
fi

if [ "$LANGUAGE" = "java" ]; then
	CC_SRC_PATH="/opt/gopath/src/github.com/chaincode/chaincode_example02/java/"
fi

echo "Channel name : "$CHANNEL_NAME

# import utils
. scripts/utils.sh

createChannel() {
	setGlobals 0 1

	if [ -z "$CORE_PEER_TLS_ENABLED" -o "$CORE_PEER_TLS_ENABLED" = "false" ]; then
                set -x
		peer channel create -o orderer.example.com:7050 -c $CHANNEL_NAME -f ./channel-artifacts/$CHANNEL_NAME/$CHANNEL_NAME.tx >&log.txt
		res=$?
                set +x
	else
				set -x
		peer channel create -o orderer.example.com:7050 -c $CHANNEL_NAME -f ./channel-artifacts/$CHANNEL_NAME/$CHANNEL_NAME.tx --tls $CORE_PEER_TLS_ENABLED --cafile $ORDERER_CA >&log.txt
		res=$?
				set +x
	fi
	cat log.txt
	verifyResult $res "Channel creation failed"
	echo "===================== Channel '$CHANNEL_NAME' created ===================== "
	echo
}

joinChannel () {
	for org in 1 2 3 4 5 6 7 8 9 10; do
	    for peer in 0; do
		joinChannelWithRetry $peer $org
		echo "===================== peer${peer}.org${org} joined channel '$CHANNEL_NAME' ===================== "
		sleep $DELAY
		echo
	    done
	done
}

## Create channel
echo "Creating channel..."
createChannel

## Join all the peers to the channel
echo "Having all peers join the channel..."
joinChannel

# Set the anchor peers for each org in the channel
echo "Updating anchor peers for org1..."
updateAnchorPeers 0 1
echo "Updating anchor peers for org2..."
updateAnchorPeers 0 2
echo "Updating anchor peers for org3..."
updateAnchorPeers 0 3
echo "Updating anchor peers for org4..."
updateAnchorPeers 0 4
echo "Updating anchor peers for org5..."
updateAnchorPeers 0 5
echo "Updating anchor peers for org6..."
updateAnchorPeers 0 6
echo "Updating anchor peers for org7..."
updateAnchorPeers 0 7
echo "Updating anchor peers for org8..."
updateAnchorPeers 0 8
echo "Updating anchor peers for org9..."
updateAnchorPeers 0 9
echo "Updating anchor peers for org10..."
updateAnchorPeers 0 10

## Install chaincode on peer0.org1 and peer0.org2
echo "Installing chaincode on peer0.org1..."
installChaincode 0 1
echo "Install chaincode on peer0.org2..."
installChaincode 0 2
echo "Installing chaincode on peer0.org3..."
installChaincode 0 3
echo "Installing chaincode on peer0.org4..."
installChaincode 0 4
echo "Instantiating chaincode on peer0.org5..."
installChaincode 0 5
echo "Installing chaincode on peer0.org6..."
installChaincode 0 6
echo "Install chaincode on peer0.org7..."
installChaincode 0 7
echo "Installing chaincode on peer0.org8..."
installChaincode 0 8
echo "Installing chaincode on peer0.org9..."
installChaincode 0 9
echo "Installing chaincode on peer0.org10..."
installChaincode 0 10

# Instantiate chaincode
echo "Instantiating chaincode on peer0.org1..."
instantiateChaincode 0 1

sleep 3

# Invoke chaincode on peer0.org1 and peer0.org2
echo "Sending invoke transaction on peer0.org1 peer0.org2..."
issueToken 0 1 0 2 0 3 0 4 0 5 0 6 0 7 0 8 0 9 0 10

echo
echo "========= All GOOD, BYFN execution completed =========== "
echo

echo
echo " _____   _   _   ____   "
echo "| ____| | \ | | |  _ \  "
echo "|  _|   |  \| | | | | | "
echo "| |___  | |\  | | |_| | "
echo "|_____| |_| \_| |____/  "
echo

exit 0
