package sqlx

import (
	. "github.com/ZenLiuCN/glu/v3"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"testing"
)

func TestSqlxHelp(t *testing.T) {
	if err := ExecuteCode(`print(help())`, 0, 0, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := ExecuteCode(`
local sqlx=require('sqlx')

for word in string.gmatch(sqlx.help(), '([^,]+)') do
	print(sqlx.help(word))
end
for word in string.gmatch(sqlx.DB.help(), '([^,]+)') do
	print(sqlx.DB.help(word))
end
for word in string.gmatch(sqlx.Tx.help(), '([^,]+)') do
	print(sqlx.Tx.help(word))
end
for word in string.gmatch(sqlx.Stmt.help(), '([^,]+)') do
	print(sqlx.Stmt.help(word))
end
for word in string.gmatch(sqlx.NamedStmt.help(), '([^,]+)') do
	print(sqlx.NamedStmt.help(word))
end
for word in string.gmatch(sqlx.Result.help(), '([^,]+)') do
	print(sqlx.Result.help(word))
end
`, 0, 0, nil, nil); err != nil {
		t.Fatal(err)
	}
}
func TestSqlx(t *testing.T) {
	if err := ExecuteCode(`
json=require "json"
sqlx=require "sqlx"
print(help())
print(sqlx.help())
print(sqlx.help('DB'))
print(sqlx.DB.help())
local db=sqlx.connect('sqlite3','file:./temp?mode=memory')
print(db:exec('create table if not exists "SOME" (ti timestamp)'):rows())
print(db:exec('insert into SOME (ti) values(\'2023-10-01\')'):rows())
print(db:exec('insert into SOME (ti) values(\'2023-10-02\')'):rows())
print(db:exec('insert into SOME (ti) values(\'2023-10-03\')'):rows())
print(db:query('select * from SOME '):json())
print(db:query('select * from SOME where ti>=?',json.parse('["2023-10-02"]')):json())
local tx=db:begin()
print('inserted in tx ',tx:exec('insert into SOME (ti) values(\'2023-10-02\')'):rows())
assert(tx:query('select * from SOME where ti=\'2023-10-02\''):size()==2)
tx:rollback()
print('rollback')
assert(db:query('select * from SOME where ti=\'2023-10-02\''):size()==1)
local tx=db:begin()
print('new tx')
print('inserts ',tx:execMany('insert into SOME(ti) values(:date)',json.parse('[{"date":"2023-11-02"},{"date":"2023-11-03"}]')))
print('queries ',tx:queryMany('select * from SOME where ti=:date',json.parse('[{"date":"2023-11-02"},{"date":"2023-11-03"}]')):json())
tx:rollback()
print('rollback')
print(db:exec('create table if not exists "SOME1" (ti number)'):rows())
db:close()
`, 0, 0, nil, nil); err != nil {
		t.Fatal(err)
	}
}
func TestSqlxPgNumeric(t *testing.T) {
	if err := ExecuteCode(`
local json=require('json')
local sqlx=require('sqlx')
local db=sqlx.connect('postgres','postgres://postgres:123456@127.0.0.1:5432/postgres?sslmode=disable')
print(db:exec('DROP TABLE IF EXISTS "SOME"'):rows())
print(db:exec('CREATE TABLE IF NOT EXISTS "SOME" (ti NUMERIC)'):rows())
print(db:exec('INSERT INTO "SOME"(ti) VALUES(1.24)'):rows())
local jo=db:query('SELECT * FROM "SOME"')
print(jo:json())
local n=sqlx.decB64(jo:get(0):get('ti'):string())
print(n)
jo:get(0):set('ti',n)
print(jo:json())
jo:set('0',json.of('{"ti":"'..n..'"}'))
print(jo:json())
db:close()
`, 0, 0, nil, nil); err != nil {
		t.Fatal(err)
	}
}
func TestMysqlNumeric(t *testing.T) {
	if err := ExecuteCode(`
local json=require('json')
local sqlx=require('sqlx')
local db=sqlx.connect('mysql','root@/profit')
print(db:exec('DROP TABLE IF EXISTS SOME'):rows())
print(db:exec('CREATE TABLE IF NOT EXISTS SOME (ti NUMERIC(10,2))'):rows())
print(db:exec('INSERT INTO SOME (ti) VALUES(1.24)'):rows())
local jo=db:query('SELECT * FROM SOME')
print(jo:json())
local n=sqlx.decB64(jo:get(0):get('ti'):string())
print(n)
jo:get(0):set('ti',n)
print(jo:json())
jo:set('0',json.of('{"ti":"'..n..'"}'))
print(jo:json())
db:close()
`, 0, 0, nil, nil); err != nil {
		t.Fatal(err)
	}
}
