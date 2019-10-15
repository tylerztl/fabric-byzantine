package mysql

import "strconv"

var (
	blockSQL    = "INSERT INTO block VALUES(?,?,?,?);"
	txSQL       = "INSERT INTO transaction VALUES(?,?,?,?);"
	blockHeight = "select max(number) as height from block;"
	blockPage   = "select * from (select number from block order by number desc limit ?,?) a left join block b on a.number = b.number;"
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
	height, err := strconv.ParseUint(string(data), 10, 64)
	if err != nil {
		panic(err.Error())
		return 0
	}
	return height
}
