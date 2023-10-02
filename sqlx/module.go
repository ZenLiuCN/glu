package sqlx

import (
	"database/sql"
	"fmt"
	. "github.com/Jeffail/gabs/v2"
	"github.com/ZenLiuCN/fn"
	. "github.com/ZenLiuCN/glu/v2"
	"github.com/ZenLiuCN/glu/v2/json"
	"github.com/jmoiron/sqlx"
	lua "github.com/yuin/gopher-lua"
)

var (
	SqlxModule        Module
	SqlxDBType        Type[*sqlx.DB]
	SqlxTxType        Type[*sqlx.Tx]
	SqlxStmtType      Type[*sqlx.Stmt]
	SqlxNamedStmtType Type[*sqlx.NamedStmt]
	SqlResultType     Type[sql.Result]
)

func init() {
	SqlxModule = NewModule(`sqlx`, `sqlx: the golang sqlx wrapper supports sqlite3 mysql and postgres database.`, true).
		AddFunc(`connect`, `connect(driver:string,dsn:string)sqlx.DB => same as sqlx.DB.new(string,string)sqlx.DB`, func(s *lua.LState) int {
			d := s.CheckString(1)
			if d == "" {
				return 0
			}
			u := s.CheckString(2)
			if u == "" {
				return 0
			}
			db, err := sqlx.Connect(d, u)
			if err != nil {
				s.RaiseError("connect error: %s", err)
				return 0
			}
			return SqlxDBType.New(s, db)
		})
	SqlxDBType = NewTypeCast[*sqlx.DB](func(a any) (v *sqlx.DB, ok bool) { v, ok = a.(*sqlx.DB); return }, `DB`, `sqlx.DB wrapper`, false, `new(driver:string,dsn:string)sqlx.DB`,
		func(s *lua.LState) (v *sqlx.DB, ok bool) {
			d := s.CheckString(1)
			if d == "" {
				return nil, false
			}
			u := s.CheckString(2)
			if u == "" {
				return nil, false
			}
			db, err := sqlx.Connect(d, u)
			if err != nil {
				s.RaiseError("connect error: %s", err)
				return nil, false
			}
			return db, true
		}).
		AddMethodCast(`query`, `query(string,Json?)json.Json =>query database`, func(s *lua.LState, data *sqlx.DB) int {
			return Raise(s, func() int {
				q, ok := CheckString(s, 2)
				if !ok {
					return 0
				}
				var r *sqlx.Rows
				var err error
				if s.GetTop() == 3 {
					if j, ok := json.JsonType.CastVar(s, 3); ok {
						if m, ok := j.Data().(map[string]any); ok {
							r, err = data.NamedQuery(q, m)
						} else if i1, ok := j.Data().([]any); ok {
							r, err = data.Queryx(q, i1...)
						} else {
							s.ArgError(3, "must a json object or json array")
							return 0
						}
					} else {
						s.ArgError(3, "must a json object or json array")
						return 0
					}
				} else if s.GetTop() != 2 {
					s.ArgError(3, fmt.Sprintf("argument error with %d args", s.GetTop()-1))
					return 0
				} else {
					r, err = data.Queryx(q)
				}
				if err != nil {
					s.RaiseError("query '%s' error :%s", q, err)
					return 0
				}
				defer r.Close()
				rs := Wrap([]any{})
				for r.Next() {
					m := make(map[string]any)
					err = r.MapScan(m)
					if err != nil {
						s.RaiseError(err.Error())
						return 0
					}
					fn.Panic(rs.ArrayAppend(m))
				}
				return json.JsonType.New(s, rs)
			})
		}).
		AddMethodCast(`exec`, `exec(string,Json?)sqlx.Result => exec SQL`, func(s *lua.LState, data *sqlx.DB) int {
			q, ok := CheckString(s, 2)
			if !ok {
				return 0
			}
			var r sql.Result
			var err error
			if s.GetTop() == 3 {
				if j, ok := json.JsonType.CastVar(s, 3); ok {
					if m, ok := j.Data().(map[string]any); ok {
						r, err = data.NamedExec(q, m)
					} else if i1, ok := j.Data().([]any); ok {
						r, err = data.Exec(q, i1...)
					} else {
						s.ArgError(3, "must a json object or json array")
						return 0
					}
				} else {
					s.ArgError(3, "must a json object or json array")
					return 0
				}
			} else if s.GetTop() != 2 {
				s.ArgError(3, fmt.Sprintf("argument error with %d args", s.GetTop()-1))
				return 0
			} else {
				r, err = data.Exec(q)
			}
			if err != nil {
				s.RaiseError(err.Error())
				return 0
			}
			return SqlResultType.New(s, r)
		}).
		AddMethodCast(`begin`, `begin()sqlx.Tx => begin transaction`, func(s *lua.LState, data *sqlx.DB) int {
			tx, err := data.Beginx()
			if err != nil {
				s.RaiseError(err.Error())
				return 0
			}
			return SqlxTxType.New(s, tx)
		}).
		AddMethodCast(`prepare`, `prepare(string)sqlx.Stmt => prepare statement`, func(s *lua.LState, data *sqlx.DB) int {
			stmt, err := data.Preparex(s.CheckString(2))
			if err != nil {
				s.RaiseError(err.Error())
				return 0
			}
			return SqlxStmtType.New(s, stmt)
		}).
		AddMethodCast(`prepareNamed`, `prepareNamed(string)sqlx.Stmt => prepare named statement`, func(s *lua.LState, data *sqlx.DB) int {
			stmt, err := data.PrepareNamed(s.CheckString(2))
			if err != nil {
				s.RaiseError(err.Error())
				return 0
			}
			return SqlxNamedStmtType.New(s, stmt)
		}).
		AddMethodCast(`close`, `close() => close database`, func(s *lua.LState, data *sqlx.DB) int {
			err := data.Close()
			if err != nil {
				s.RaiseError(`close database: %s`, err)
			}
			return 0
		})
	SqlResultType = NewTypeCast[sql.Result](func(a any) (v sql.Result, ok bool) { v, ok = a.(sql.Result); return }, `Result`, `sql.Result wrap`, false, ``, nil).
		AddMethodCast(`lastID`, `lastID()number => last inserted id or raise error`, func(s *lua.LState, data sql.Result) int {
			v, err := data.LastInsertId()
			if err != nil {
				s.RaiseError("error %s", err)
				return 0
			}
			s.Push(lua.LNumber(v))
			return 1
		}).
		AddMethodCast(`rows`, `rows()number =>affected rows or raise error`, func(s *lua.LState, data sql.Result) int {
			v, err := data.RowsAffected()
			if err != nil {
				s.RaiseError("error %s", err)
				return 0
			}
			s.Push(lua.LNumber(v))
			return 1
		})
	SqlxTxType = NewTypeCast(func(a any) (v *sqlx.Tx, ok bool) { v, ok = a.(*sqlx.Tx); return }, `Tx`, `sqlx.Tx wrapper`, false, `none`, nil).
		AddMethodCast(`commit`, `commit()`, func(s *lua.LState, data *sqlx.Tx) int {
			err := data.Commit()
			if err != nil {
				s.RaiseError(err.Error())
			}
			return 0
		}).
		AddMethodCast(`rollback`, `rollback()`, func(s *lua.LState, data *sqlx.Tx) int {
			err := data.Rollback()
			if err != nil {
				s.RaiseError(err.Error())
			}
			return 0
		}).
		AddMethodCast(`exec`, `exec(string,Json?)sqlx.Result`, func(s *lua.LState, data *sqlx.Tx) int {
			q, ok := CheckString(s, 2)
			if !ok {
				return 0
			}
			var r sql.Result
			var err error
			if s.GetTop() == 3 {
				if j, ok := json.JsonType.CastVar(s, 3); ok {
					if m, ok := j.Data().(map[string]any); ok {
						r, err = data.NamedExec(q, m)
					} else if i1, ok := j.Data().([]any); ok {
						r, err = data.Exec(q, i1...)
					} else {
						s.ArgError(3, "must a json object or json array")
						return 0
					}
				} else {
					s.ArgError(3, "must a json object or json array")
					return 0
				}
			} else if s.GetTop() != 2 {
				s.ArgError(3, fmt.Sprintf("argument error with %d args", s.GetTop()-1))
				return 0
			} else {
				r, err = data.Exec(q)
			}
			if err != nil {
				s.RaiseError("exec error :%s", err)
				return 0
			}
			return SqlResultType.New(s, r)

		}).
		AddMethodCast(`query`, `query(string,Json?)sqlx.Result`, func(s *lua.LState, data *sqlx.Tx) int {
			q, ok := CheckString(s, 2)
			if !ok {
				return 0
			}
			var r *sqlx.Rows
			var err error
			if s.GetTop() == 3 {
				if j, ok := json.JsonType.CastVar(s, 3); ok {
					if m, ok := j.Data().(map[string]any); ok {
						r, err = data.NamedQuery(q, m)
					} else if i1, ok := j.Data().([]any); ok {
						r, err = data.Queryx(q, i1...)
					} else {
						s.ArgError(3, "must a json object or json array")
						return 0
					}
				} else {
					s.ArgError(3, "must a json object or json array")
					return 0
				}
			} else if s.GetTop() != 2 {
				s.ArgError(3, fmt.Sprintf("argument error with %d args", s.GetTop()-1))
				return 0
			} else {
				r, err = data.Queryx(q)
			}
			if err != nil {
				s.RaiseError("query '%s' error :%s", q, err)
				return 0
			}
			defer r.Close()
			rs := Wrap([]any{})
			for r.Next() {
				m := make(map[string]any)
				err = r.MapScan(m)
				if err != nil {
					s.RaiseError("query error :%s", err)
					return 0
				}
				fn.Panic(rs.ArrayAppend(m))
			}
			return json.JsonType.New(s, rs)

		})
	SqlxStmtType = NewTypeCast(func(a any) (v *sqlx.Stmt, ok bool) { v, ok = a.(*sqlx.Stmt); return }, `Stmt`, `sqlx.Stmt wrapper`, false, `none`, nil).
		AddMethodCast(`query`, `query(Json?)Json => param must a json Array`, func(s *lua.LState, data *sqlx.Stmt) int {
			return Raise(s, func() int {
				var r *sqlx.Rows
				var err error
				if s.GetTop() == 2 {
					if j, ok := json.JsonType.CastVar(s, 2); ok {
						if i1, ok := j.Data().([]any); ok {
							r, err = data.Queryx(i1...)
						} else {
							s.ArgError(2, "must a json object or json array")
							return 0
						}
					} else {
						s.ArgError(2, "must a json object or json array")
						return 0
					}
				} else if s.GetTop() != 2 {
					s.ArgError(2, fmt.Sprintf("argument error with %d args", s.GetTop()-1))
					return 0
				} else {
					r, err = data.Queryx()
				}
				if err != nil {
					s.RaiseError("query error :%s", err)
					return 0
				}
				defer r.Close()
				rs := Wrap([]any{})
				for r.Next() {
					m := make(map[string]any)
					err = r.MapScan(m)
					if err != nil {
						s.RaiseError(err.Error())
						return 0
					}
					fn.Panic(rs.ArrayAppend(m))
				}
				return json.JsonType.New(s, rs)
			})
		}).
		AddMethodCast(`exec`, `exec(Json?)sqlx.Result => param must a json Array`, func(s *lua.LState, data *sqlx.Stmt) int {
			return Raise(s, func() int {
				var r sql.Result
				var err error
				if s.GetTop() == 2 {
					if j, ok := json.JsonType.CastVar(s, 2); ok {
						if i1, ok := j.Data().([]any); ok {
							r, err = data.Exec(i1...)
						} else {
							s.ArgError(2, "must a json object or json array")
							return 0
						}
					} else {
						s.ArgError(2, "must a json object or json array")
						return 0
					}
				} else if s.GetTop() != 2 {
					s.ArgError(3, fmt.Sprintf("argument error with %d args", s.GetTop()-1))
					return 0
				} else {
					r, err = data.Exec()
				}
				if err != nil {
					s.RaiseError(err.Error())
					return 0
				}
				return SqlResultType.New(s, r)
			})
		})
	SqlxNamedStmtType = NewTypeCast(func(a any) (v *sqlx.NamedStmt, ok bool) { v, ok = a.(*sqlx.NamedStmt); return }, `NamedStmt`, `sqlx.NamedStmt wrapper`, false, `none`, nil).
		AddMethodCast(`query`, `query(Json)Json => param must a json Object`, func(s *lua.LState, data *sqlx.NamedStmt) int {
			return Raise(s, func() int {
				var r *sqlx.Rows
				var err error
				if s.GetTop() > 2 {
					s.ArgError(3, fmt.Sprintf("argument error with %d args", s.GetTop()-1))
					return 0
				}
				if j, ok := json.JsonType.CastVar(s, 2); ok {
					if m, ok := j.Data().(map[string]any); ok {
						r, err = data.Queryx(m)
					} else {
						s.ArgError(2, "must a json object")
						return 0
					}
				} else {
					s.ArgError(2, "must a json object")
					return 0
				}
				if err != nil {
					s.RaiseError("query error :%s", err)
					return 0
				}
				defer r.Close()
				rs := Wrap([]any{})
				for r.Next() {
					m := make(map[string]any)
					err = r.MapScan(m)
					if err != nil {
						s.RaiseError(err.Error())
						return 0
					}
					fn.Panic(rs.ArrayAppend(m))
				}
				return json.JsonType.New(s, rs)
			})
		}).
		AddMethodCast(`exec`, `exec(Json)sqlx.Result => param must a json Object`, func(s *lua.LState, data *sqlx.NamedStmt) int {
			return Raise(s, func() int {
				var r sql.Result
				var err error
				if s.GetTop() > 2 {
					s.ArgError(3, fmt.Sprintf("argument error with %d args", s.GetTop()-1))
					return 0
				}
				if j, ok := json.JsonType.CastVar(s, 2); ok {
					if m, ok := j.Data().(map[string]any); ok {
						r, err = data.Exec(m)
					} else {
						s.ArgError(2, "must a json object")
						return 0
					}
				} else {
					s.ArgError(2, "must a json object")
					return 0
				}
				if err != nil {
					s.RaiseError(err.Error())
					return 0
				}
				return SqlResultType.New(s, r)
			})
		})

	SqlxModule.AddModule(SqlxDBType)
	SqlxModule.AddModule(SqlxStmtType)
	SqlxModule.AddModule(SqlxNamedStmtType)
	SqlxModule.AddModule(SqlResultType)
	SqlxModule.AddModule(SqlxTxType)
	fn.Panic(Register(SqlxModule))
}
