package mysql

var (
	blockSQL  = "INSERT INTO block VALUES(?,?,?,?);"
	txSQL     = "INSERT INTO transaction VALUES(?,?,?,?);"
	blockPage = "select * from (select number from block order by number desc limit ?,?) a left join block b on a.number = b.number;"
)

func BlockPage(pageId, size int) ([]byte, error) {
	return dbMgr.QueryRows(blockPage, (pageId-1)*size, size)
}
