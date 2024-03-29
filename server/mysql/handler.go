package mysql

import "strconv"

var (
	blockSQL    = "INSERT INTO block VALUES(?,?,?,?,?,?,?);"
	txSQL       = "INSERT INTO transaction VALUES(?,?,?,?,?,?,?);"
	blockHeight = "select max(number) as height from block;"
	blockPage   = "select * from (select number from block order by number desc limit ?,?) a left join block b on a.number = b.number;"
	txPage      = "select * from (select tx_index from transaction order by tx_index desc limit ?,?) a left join transaction b on a.tx_index = b.tx_index;"
	updateTX    = "UPDATE transaction SET peer=?,tx_type=? WHERE tx_id=?;"
	queryTX     = "select * from transaction where tx_id=?;"
	peerList    = "select * from peer;"
	updatePeer    = "UPDATE peer SET peer_type=? WHERE name=?;"
	txNumber    = "select sum(tx_count) as tx from block;"
)

func BlockPage(pageId, size int) ([]byte, error) {
	return GetDBMgr().QueryRows(blockPage, (pageId-1)*size, size)
}

func GetBlockHeight() uint64 {
	data, err := GetDBMgr().QueryValue(blockHeight)
	if err != nil {
		panic(err.Error())
		return 0
	}
	height, _ := strconv.ParseUint(string(data), 10, 64)
	return height
}

func TransactionPage(pageId, size int) ([]byte, error) {
	return GetDBMgr().QueryRows(txPage, (pageId-1)*size, size)
}

func QueryTransaction(txId string) ([]byte, error) {
	return GetDBMgr().QueryRows(queryTX, txId)
}

func UpdateTransaction(peer, txId string, txType int) error {
	return GetDBMgr().InsertOrUpdate(updateTX, peer, txType, txId)
}

func PeerList() ([]byte, error) {
	return GetDBMgr().QueryRows(peerList)
}

func UpdatePeers(peer string, peerType int) error {
	return GetDBMgr().InsertOrUpdate(updatePeer, peerType, peer)
}

func TxNumber() uint64 {
	data, err := GetDBMgr().QueryValue(txNumber)
	if err != nil {
		panic(err.Error())
		return 0
	}
	height, _ := strconv.ParseUint(string(data), 10, 64)
	return height
}

func UpdateBlock(peer, txId string, txType int) error {
	return GetDBMgr().InsertOrUpdate(updateTX, peer, txType, txId)
}