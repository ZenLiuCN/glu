package sqlx

import (
	. "github.com/ZenLiuCN/glu/v2"
	_ "github.com/mattn/go-sqlite3"
	"testing"
)

func TestSqlx(t *testing.T) {
	if err := ExecuteCode(`print(help())`, 0, 0, nil, nil); err != nil {
		t.Fatal(err)
	}

	if err := ExecuteCode(`
local json=require('json')
local sqlx=require('sqlx')
print(sqlx.help())
print(sqlx.help('DB'))
print(sqlx.DB.help())
local db=sqlx.connect('sqlite3','file:./temp?mode=memory')
local r=db:exec('create table if not exists "SOME" (ti timestamp)')
print(r:rows())
print(db:exec('insert into SOME (ti) values(\'2023-10-01\')'):rows())
print(db:exec('insert into SOME (ti) values(\'2023-10-02\')'):rows())
print(db:exec('insert into SOME (ti) values(\'2023-10-03\')'):rows())
print(db:query('select * from SOME '):json())
print(db:query('select * from SOME where ti>=?',json.of('["2023-10-02"]')):json())
local tx=db:begin()
print('inserted in tx ',tx:exec('insert into SOME (ti) values(\'2023-10-02\')'):rows())
assert(tx:query('select * from SOME where ti=\'2023-10-02\''):size()==2)
tx:rollback()
print('rollback')
assert(db:query('select * from SOME where ti=\'2023-10-02\''):size()==1)
local tx=db:begin()
print('new tx')
print('inserts ',tx:execMany('insert into SOME(ti) values(:date)',json.of('[{"date":"2023-11-02"},{"date":"2023-11-03"}]')))
print('queries ',tx:queryMany('select * from SOME where ti=:date',json.of('[{"date":"2023-11-02"},{"date":"2023-11-03"}]')):json())
tx:rollback()
print('rollback')
db:close()
`, 0, 0, nil, nil); err != nil {
		t.Fatal(err)
	}
}
