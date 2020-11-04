package mysqlkit

import (
	"bytes"
	"database/sql"
	"errors"
	_ "github.com/go-sql-driver/mysql"
	"github.com/rz1226/simplegokit/blackboardkit"
	"github.com/rz1226/simplegokit/orm"
	"reflect"
	"strings"
	"time"
)

//以前放在utilx2的那个mysql操作库，看上好像比较灵活，实际上非常难用。心智负担较高

type MysqlKit struct {
	realPool  *sql.DB
	connected bool
	conerr    error
	bb        *blackboardkit.BlackBoradKit
}

func NewMysqlKit(conStr string, maxOpenConns int) *MysqlKit {
	p := &MysqlKit{}
	p.bb = blackboardkit.NewBlockBorad()
	p.bb.InitLogKit("mysql_error", "mysql_info")
	p.bb.SetLogReadme("mysql_info", "mysql执行中的日志信息")
	p.bb.SetLogReadme("mysql_error", "mysql执行中的错误信息")

	p.bb.InitTimerKit("mysql_timer")
	p.bb.SetTimerReadme("mysql_timer", "mysql客户端耗时记录")
	p.bb.SetName("MySql客户端日志汇集")
	p.bb.Ready()

	realPool, err := sql.Open("mysql", conStr)
	p.conerr = err
	if err == nil {
		realPool.SetMaxOpenConns(maxOpenConns)
		realPool.SetMaxIdleConns(10)
		realPool.SetConnMaxLifetime(time.Second * 10000)
		p.realPool = realPool
		p.connected = true
	} else {
		p.connected = false
	}
	return p
}

//获取*sql.DB
func (p *MysqlKit) DB() *sql.DB {
	return p.realPool
}

/*
var d []*stru
err  := MysqlKit.Query(sql,id).Get(&d)

*/
//

func (p *MysqlKit) Query(sql string, args ...interface{}) *RowsKit {
	if p.connected == false {
		return &RowsKit{r: nil, err: p.conerr}
	}
	t := p.bb.Start("mysql_timer", sql)
	rows, err := p.realPool.Query(sql, args...)
	if err != nil {
		p.bb.Log("mysql_error", err, sql, args)
		return &RowsKit{r: nil, err: err}
	}
	p.bb.End(t)
	p.bb.Log("mysql_info", sql, args)
	return &RowsKit{r: rows, err: nil}
}

type RowsKit struct {
	r   *sql.Rows
	err error
}

func (r *RowsKit) Err() error {
	return r.err
}

//对查询结果单纯的close
func (r *RowsKit) Close() {
	defer r.r.Close()
}
func (r *RowsKit) GetStrus(st interface{}) error {
	if r.err != nil {
		return r.err
	}
	defer r.r.Close()
	err := orm.Rows2Strus(r.r, st)
	if err != nil {
		return err
	}
	return nil
}
func (r *RowsKit) GetStru(st interface{}) error {
	if r.err != nil {
		return r.err
	}
	defer r.r.Close()
	err := orm.Rows2Stru(r.r, st)
	if err != nil {
		return err
	}
	return nil
}
func (r *RowsKit) GetNs(st interface{}) error {
	if r.err != nil {
		return r.err
	}
	defer r.r.Close()
	err := orm.Rows2Cnts(r.r, st)
	if err != nil {
		return err
	}
	return nil
}
func (r *RowsKit) GetN(st interface{}) error {
	if r.err != nil {
		return r.err
	}
	defer r.r.Close()
	err := orm.Rows2Cnt(r.r, st)
	if err != nil {
		return err
	}
	return nil
}

func (p *MysqlKit) Ex(sqlStr string, args ...interface{}) (int64, error) {
	return p.Exec(sqlStr, args)
}

// data 是一个slice, 里面的个数对应sqlStr里面？的数量
func (p *MysqlKit) Exec(sqlStr string, data []interface{}) (int64, error) {
	if p.connected == false {
		return 0, p.conerr
	}
	db := p.realPool
	//插入数据
	stmt, err := db.Prepare(sqlStr)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()
	length := len(data)
	if err != nil {
		return 0, err
	}

	fn := reflect.ValueOf(stmt.Exec)
	fnParams := make([]reflect.Value, length)
	for i := 0; i < length; i++ {
		fnParams[i] = reflect.ValueOf(data[i])
	}
	t := p.bb.Start("mysql_timer", sqlStr)
	callResult := fn.Call(fnParams)
	p.bb.End(t)
	if callResult[1].Interface() != nil {
		p.bb.Log("mysql_error", callResult[1].Interface().(error), sqlStr, data)
		return 0, callResult[1].Interface().(error)
	}
	p.bb.Log("mysql_info", sqlStr, data)
	result := callResult[0].Interface().(sql.Result)
	if isUpdate(sqlStr) || isDelete(sqlStr) {
		return result.RowsAffected() //本身就是多个返回值
	}
	if isInsert(sqlStr) {
		return result.LastInsertId() //本身就是多个返回值
	}
	if isReplace(sqlStr) {
		return result.LastInsertId()
	}
	if isTruncate(sqlStr) {
		return 0, nil
	}

	return 0, errors.New("only support update insert delete replace")
}
func isReplace(sqlStr string) bool {
	str := strings.TrimSpace(strings.ToLower(sqlStr))
	if strings.HasPrefix(str, "replace") {
		return true
	}
	return false
}

func isInsert(sqlStr string) bool {
	str := strings.TrimSpace(strings.ToLower(sqlStr))
	if strings.HasPrefix(str, "insert") {
		return true
	}
	return false
}

func isUpdate(sqlStr string) bool {
	str := strings.TrimSpace(strings.ToLower(sqlStr))
	if strings.HasPrefix(str, "update") {
		return true
	}
	return false
}

func isDelete(sqlStr string) bool {
	str := strings.TrimSpace(strings.ToLower(sqlStr))
	if strings.HasPrefix(str, "delete") {
		return true
	}
	return false
}
func isTruncate(sqlStr string) bool {
	str := strings.TrimSpace(strings.ToLower(sqlStr))
	if strings.HasPrefix(str, "truncate") {
		return true
	}
	return false
}

/*************************组成批量sql插入语句*********************************/

//生成用来批量插入的参数
func BatchInsertParams(datas [][]interface{}) (string, []interface{}) {
	sqlParams := make([]interface{}, 0, 200)
	sqlStringBuffer := bytes.Buffer{}
	for _, oneData := range datas {
		length := len(oneData)
		if length == 0 {
			continue
		}
		sqlStringBuffer.WriteString("(")
		for idx, val := range oneData {
			sqlParams = append(sqlParams, val)
			if length == idx+1 {
				sqlStringBuffer.WriteString("?")
			} else {
				sqlStringBuffer.WriteString("?,")
			}
		}
		sqlStringBuffer.WriteString("),")
	}
	sql := strings.TrimRight(sqlStringBuffer.String(), ",")
	return sql, sqlParams
}
