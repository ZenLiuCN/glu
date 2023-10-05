package sqlx

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	. "github.com/Jeffail/gabs/v2"
	"github.com/ZenLiuCN/fn"
	. "github.com/ZenLiuCN/glu/v3"
	"github.com/ZenLiuCN/glu/v3/json"
	"github.com/jmoiron/sqlx"
	lua "github.com/yuin/gopher-lua"
)

var (
	MODULE    Module
	DB        Type[*sqlx.DB]
	TX        Type[*sqlx.Tx]
	Stmt      Type[*sqlx.Stmt]
	NamedStmt Type[*sqlx.NamedStmt]
	Result    Type[sql.Result]
)

func init() {
	MODULE = NewModule(`sqlx`, `sqlx: the golang sqlx`, true).
		AddFunc(`connect`, `(driver:string,dsn:string)DB 	 same as sqlx.DB.new(string,string)DB`, func(s *lua.LState) int {
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
			return DB.New(s, db)
		}).
		AddFunc(`decB64`, `(string)string 	 decode base64 to string`, func(s *lua.LState) int {
			d := s.CheckString(1)
			if d == "" {
				return 0
			}
			b, err := base64.StdEncoding.DecodeString(d)
			if err != nil {
				s.RaiseError(err.Error())
				return 0
			}
			s.Push(lua.LString(b))
			return 1
		}).
		AddFunc(`encB64`, `(string)string 	 encode string to base64`, func(s *lua.LState) int {
			d := s.CheckString(1)
			if d == "" {
				return 0
			}
			b := base64.StdEncoding.EncodeToString([]byte(d))
			s.Push(lua.LString(b))
			return 1
		}).
		AddFunc(`to_num`, `(JSON,string ...)JSON 	 convert json array fields from base64 to numeric string`, func(s *lua.LState) int {
			g := json.JSON.Check(s, 1)
			if _, ok := g.Data().([]any); !ok {
			}
			n := s.GetTop() - 2
			if n < 0 {
				s.RaiseError(`must have field names`)
			}
			names := make([]string, 0, n)
			for i := 0; i <= n; i++ {
				sx := s.CheckString(i + 2)
				if sx == "" {
					return 0
				}
				names = append(names, sx)
			}
			size := fn.Panic1(g.ArrayCount())
			for i := 0; i < size; i++ {
				val := g.Index(i)
				for _, name := range names {
					v := val.Path(name)
					if v != nil {
						b64 := v.String()
						b64 = b64[1 : len(b64)-1]
						b64 = string(fn.Panic1(base64.StdEncoding.DecodeString(b64)))
						fn.Panic1(val.Set(b64, name))
					}
				}
			}
			return json.JSON.New(s, g)
		}).
		AddFunc(`from_num`, `(JSON,string ...)JSON 	 convert json array fields from numeric string to base64`, func(s *lua.LState) int {
			g := json.JSON.Check(s, 1)
			if _, ok := g.Data().([]any); !ok {
				s.RaiseError(`must a json array`)
				return 0
			}
			n := s.GetTop() - 2
			if n < 0 {
				s.RaiseError(`must have field names`)
				return 0
			}
			names := make([]string, 0, n)
			for i := 0; i <= n; i++ {
				sx := s.CheckString(i + 2)
				names = append(names, sx)
			}
			size := fn.Panic1(g.ArrayCount())
			for i := 0; i < size; i++ {
				val := g.Index(i)
				for _, name := range names {
					v := val.Path(name)
					if v != nil {
						b64 := v.Data().(string)
						fn.Panic1(val.Set(base64.StdEncoding.EncodeToString([]byte(b64)), name))
					}
				}
			}
			return json.JSON.New(s, g)
		})
	DB = NewTypeCast[*sqlx.DB](func(a any) (v *sqlx.DB, ok bool) { v, ok = a.(*sqlx.DB); return }, `DB`, `sqlx.DB`, false, `new(string,string)DB`,
		func(s *lua.LState) (v *sqlx.DB) {
			d := s.CheckString(1)
			u := s.CheckString(2)
			db, err := sqlx.Connect(d, u)
			if err != nil {
				s.RaiseError("connect error: %s", err)
			}
			return db
		}).
		AddMethodCast(`query`, `(string,JSON?)JSON		query database`, func(s *lua.LState, data *sqlx.DB) int {
			return Raise(s, func() int {
				q := CheckString(s, 2)
				var r *sqlx.Rows
				var err error
				if s.GetTop() == 3 {
					j := json.JSON.Check(s, 3)
					if m, ok := j.Data().(map[string]any); ok {
						r, err = data.NamedQuery(q, m)
					} else if i1, ok := j.Data().([]any); ok {
						r, err = data.Queryx(q, i1...)
					} else {
						s.ArgError(3, "must a json object or json array")

					}

				} else if s.GetTop() != 2 {
					s.ArgError(3, fmt.Sprintf("argument error with %d args", s.GetTop()-1))

				} else {
					r, err = data.Queryx(q)
				}
				if err != nil {
					s.RaiseError("query '%s' error :%s", q, err)

				}
				defer r.Close()
				rs := Wrap([]any{})
				for r.Next() {
					m := make(map[string]any)
					err = r.MapScan(m)
					if err != nil {
						s.RaiseError(err.Error())
					}
					fn.Panic(rs.ArrayAppend(m))
				}
				return json.JSON.New(s, rs)
			})
		}).
		AddMethodCast(`exec`, `(string,JSON?)Result		exec SQL`, func(s *lua.LState, data *sqlx.DB) int {
			q := CheckString(s, 2)
			var r sql.Result
			var err error
			if s.GetTop() == 3 {
				j := json.JSON.Check(s, 3)
				if m, ok := j.Data().(map[string]any); ok {
					r, err = data.NamedExec(q, m)
				} else if i1, ok := j.Data().([]any); ok {
					r, err = data.Exec(q, i1...)
				} else {
					s.ArgError(3, "must a json object or json array")
				}
			} else if s.GetTop() != 2 {
				s.ArgError(3, fmt.Sprintf("argument error with %d args", s.GetTop()-1))
			} else {
				r, err = data.Exec(q)
			}
			if err != nil {
				s.RaiseError(err.Error())
			}
			return Result.New(s, r)
		}).
		AddMethodCast(`queryMany`, `(string,JSON)JSON		query many from json array of named or array parameters`, func(s *lua.LState, data *sqlx.DB) int {
			return Raise(s, func() int {
				q := CheckString(s, 2)
				if s.GetTop() != 3 {
					s.RaiseError("queries(SQL,array of object or array)")
				}
				j := json.JSON.Check(s, 3)
				if i1, ok := j.Data().([]any); ok {
					rs := Wrap(make([]any, 0, 1))
					for iy, val := range i1 {
						var err error
						var r *sqlx.Rows
						if m, ok := val.(map[string]any); ok {
							r, err = data.NamedQuery(q, m)
						} else if m, ok := val.([]any); ok {
							r, err = data.Queryx(q, m...)
						} else {
							s.RaiseError("require json array of object or array which is not at %d", iy)
						}
						if err != nil {
							s.RaiseError("process %d: %s", iy, err.Error())
						}
						for r.Next() {
							m := make(map[string]any)
							if err = r.MapScan(m); err != nil {
								s.RaiseError(err.Error())
								_ = r.Close()
							}
							if err = rs.ArrayAppend(m); err != nil {
								s.RaiseError(err.Error())
								_ = r.Close()
							}
						}
						_ = r.Close()
					}
					return json.JSON.New(s, rs)
				}
				s.ArgError(3, "must a json array of object")
				return 0
			})

		}).
		AddMethodCast(`execMany`, `(string,JSON)number		exec SQL with json array of named or array parameters`, func(s *lua.LState, data *sqlx.DB) int {
			return Raise(s, func() int {
				q := CheckString(s, 2)
				if s.GetTop() != 3 {
					s.RaiseError("execs(SQL,JsonArrayOfNamedParameters)")
				}
				j := json.JSON.Check(s, 3)
				if i1, ok := j.Data().([]any); ok {
					n := int64(0)
					tx := data.MustBegin()
					for iy, val := range i1 {
						var err error
						var r sql.Result
						if m, ok := val.(map[string]any); ok {
							r, err = data.NamedExec(q, m)
						} else if m, ok := val.([]any); ok {
							r, err = data.Exec(q, m...)
						} else {
							s.RaiseError("require json array of object or array which is not at %d", iy)
						}
						if err != nil {
							s.RaiseError("process %d: %s", iy, err.Error())
						}
						n += fn.Panic1(r.RowsAffected())
					}
					fn.Panic(tx.Commit())
					s.Push(lua.LNumber(n))
					return 1
				}
				s.ArgError(3, "must a json array of objects")
				return 0
			})
		}).
		AddMethodCast(`begin`, `()Tx		begin transaction`, func(s *lua.LState, data *sqlx.DB) int {
			tx, err := data.Beginx()
			if err != nil {
				s.RaiseError(err.Error())
			}
			return TX.New(s, tx)
		}).
		AddMethodCast(`prepare`, `(string)Stmt		prepare statement`, func(s *lua.LState, data *sqlx.DB) int {
			stmt, err := data.Preparex(s.CheckString(2))
			if err != nil {
				s.RaiseError(err.Error())
			}
			return Stmt.New(s, stmt)
		}).
		AddMethodCast(`prepareNamed`, `(string)NamedStmt		prepare named statement`, func(s *lua.LState, data *sqlx.DB) int {
			stmt, err := data.PrepareNamed(s.CheckString(2))
			if err != nil {
				s.RaiseError(err.Error())
			}
			return NamedStmt.New(s, stmt)
		}).
		AddMethodCast(`close`, `() 	 close database`, func(s *lua.LState, data *sqlx.DB) int {
			err := data.Close()
			if err != nil {
				s.RaiseError(`close database: %s`, err)
			}
			return 0
		})

	Result = NewTypeCast[sql.Result](func(a any) (v sql.Result, ok bool) { v, ok = a.(sql.Result); return }, `Result`, `sql.Result`, false, ``, nil).
		AddMethodCast(`lastID`, `()number 	 last inserted id or raise error`, func(s *lua.LState, data sql.Result) int {
			v, err := data.LastInsertId()
			if err != nil {
				s.RaiseError("error %s", err)
			}
			s.Push(lua.LNumber(v))
			return 1
		}).
		AddMethodCast(`rows`, `()number 	affected rows or raise error`, func(s *lua.LState, data sql.Result) int {
			v, err := data.RowsAffected()
			if err != nil {
				s.RaiseError("error %s", err)
			}
			s.Push(lua.LNumber(v))
			return 1
		})

	TX = NewTypeCast(func(a any) (v *sqlx.Tx, ok bool) { v, ok = a.(*sqlx.Tx); return }, `Tx`, `sqlx.Tx`, false, `none`, nil).
		AddMethodCast(`exec`, `(string,JSON?)Result			execute command`, func(s *lua.LState, data *sqlx.Tx) int {
			q := CheckString(s, 2)
			var r sql.Result
			var err error
			if s.GetTop() == 3 {
				j := json.JSON.Check(s, 3)
				if m, ok := j.Data().(map[string]any); ok {
					r, err = data.NamedExec(q, m)
				} else if i1, ok := j.Data().([]any); ok {
					r, err = data.Exec(q, i1...)
				} else {
					s.ArgError(3, "must a json object or json array")
				}
			} else if s.GetTop() != 2 {
				s.ArgError(3, fmt.Sprintf("argument error with %d args", s.GetTop()-1))
			} else {
				r, err = data.Exec(q)
			}
			if err != nil {
				s.RaiseError(err.Error())
			}
			return Result.New(s, r)
		}).
		AddMethodCast(`query`, `(string,JSON?)Result		query database`, func(s *lua.LState, data *sqlx.Tx) int {
			q := CheckString(s, 2)
			var r *sqlx.Rows
			var err error
			if s.GetTop() == 3 {
				j := json.JSON.Check(s, 3)
				if m, ok := j.Data().(map[string]any); ok {
					r, err = data.NamedQuery(q, m)
				} else if i1, ok := j.Data().([]any); ok {
					r, err = data.Queryx(q, i1...)
				} else {
					s.ArgError(3, "must a json object or json array")
				}
			} else if s.GetTop() != 2 {
				s.ArgError(3, fmt.Sprintf("argument error with %d args", s.GetTop()-1))
			} else {
				r, err = data.Queryx(q)
			}
			if err != nil {
				s.RaiseError("query '%s' error :%s", q, err)
			}
			defer r.Close()
			rs := Wrap([]any{})
			for r.Next() {
				m := make(map[string]any)
				err = r.MapScan(m)
				if err != nil {
					s.RaiseError("query error :%s", err)
				}
				fn.Panic(rs.ArrayAppend(m))
			}
			return json.JSON.New(s, rs)

		}).
		AddMethodCast(`queryMany`, `(string,JSON)JSON 		query many from json array with named parameters`, func(s *lua.LState, data *sqlx.Tx) int {
			return Raise(s, func() int {
				q := CheckString(s, 2)
				if s.GetTop() != 3 {
					s.RaiseError("queries(SQL,JsonArrayOfNamedParameters)")
					return 0
				}
				j := json.JSON.Check(s, 3)
				if i1, ok := j.Data().([]any); ok {
					rs := Wrap(make([]any, 0, 1))
					for iy, val := range i1 {
						if m, ok := val.(map[string]any); !ok {
							s.RaiseError("require json array of objects which is not at %d", iy)
							return 0
						} else {
							r, err := data.NamedQuery(q, m)
							if err != nil {
								s.RaiseError("process %d: %s", iy, err.Error())
							}
							for r.Next() {
								m := make(map[string]any)
								if err = r.MapScan(m); err != nil {
									s.RaiseError(err.Error())
									_ = r.Close()
								}
								if err = rs.ArrayAppend(m); err != nil {
									s.RaiseError(err.Error())
									_ = r.Close()
								}
							}
							_ = r.Close()
						}
					}
					return json.JSON.New(s, rs)
				}
				s.ArgError(3, "must a json array of objects")
				return 0
			})

		}).
		AddMethodCast(`execMany`, `(string,JSON)number		exec SQL with json array of named parameters`, func(s *lua.LState, data *sqlx.Tx) int {
			return Raise(s, func() int {
				q := CheckString(s, 2)
				if s.GetTop() != 3 {
					s.RaiseError("execs(SQL,JsonArrayOfNamedParameters)")
				}
				j := json.JSON.Check(s, 3)
				if i1, ok := j.Data().([]any); ok {
					n := int64(0)
					for iy, val := range i1 {
						if m, ok := val.(map[string]any); !ok {
							s.RaiseError("require json array of objects which is not at %d", iy)
						} else {
							r, err := data.NamedExec(q, m)
							if err != nil {
								s.RaiseError("process %d: %s", iy, err.Error())
							}
							n += fn.Panic1(r.RowsAffected())
						}
					}
					s.Push(lua.LNumber(n))
					return 1
				}

				s.ArgError(3, "must a json array of objects")
				return 0
			})
		}).
		AddMethodCast(`prepare`, `(string)Stmt		prepare statement`, func(s *lua.LState, data *sqlx.Tx) int {
			stmt, err := data.Preparex(s.CheckString(2))
			if err != nil {
				s.RaiseError(err.Error())
				return 0
			}
			return Stmt.New(s, stmt)
		}).
		AddMethodCast(`prepareNamed`, `(string)NameStmt		prepare named statement`, func(s *lua.LState, data *sqlx.Tx) int {
			stmt, err := data.PrepareNamed(s.CheckString(2))
			if err != nil {
				s.RaiseError(err.Error())
			}
			return NamedStmt.New(s, stmt)
		}).
		AddMethodCast(`commit`, `()		commit transaction`, func(s *lua.LState, data *sqlx.Tx) int {
			err := data.Commit()
			if err != nil {
				s.RaiseError(err.Error())
			}
			return 0
		}).
		AddMethodCast(`rollback`, `()	rollback transaction`, func(s *lua.LState, data *sqlx.Tx) int {
			err := data.Rollback()
			if err != nil {
				s.RaiseError(err.Error())
			}
			return 0
		})

	Stmt = NewTypeCast(func(a any) (v *sqlx.Stmt, ok bool) { v, ok = a.(*sqlx.Stmt); return }, `Stmt`, `sqlx.Stmt`, false, `none`, nil).
		AddMethodCast(`query`, `(JSON?)JSON 	 param must a json Array`, func(s *lua.LState, data *sqlx.Stmt) int {
			return Raise(s, func() int {
				var r *sqlx.Rows
				var err error
				if s.GetTop() == 2 {
					j := json.JSON.Check(s, 2)
					if i1, ok := j.Data().([]any); ok {
						r, err = data.Queryx(i1...)
					} else {
						s.ArgError(2, "must a json object or json array")
					}
				} else if s.GetTop() != 2 {
					s.ArgError(3, fmt.Sprintf("argument error with %d args", s.GetTop()-1))
				} else {
					r, err = data.Queryx()
				}
				if err != nil {
					s.RaiseError("query error :%s", err)
				}
				defer r.Close()
				rs := Wrap([]any{})
				for r.Next() {
					m := make(map[string]any)
					err = r.MapScan(m)
					if err != nil {
						s.RaiseError(err.Error())
					}
					fn.Panic(rs.ArrayAppend(m))
				}
				return json.JSON.New(s, rs)
			})
		}).
		AddMethodCast(`exec`, `(JSON?)Result 	 param must a json Array`, func(s *lua.LState, data *sqlx.Stmt) int {
			return Raise(s, func() int {
				var r sql.Result
				var err error
				if s.GetTop() == 2 {
					j := json.JSON.Check(s, 2)
					if i1, ok := j.Data().([]any); ok {
						r, err = data.Exec(i1...)
					} else {
						s.ArgError(2, "must a json object or json array")
					}

				} else if s.GetTop() != 2 {
					s.ArgError(3, fmt.Sprintf("argument error with %d args", s.GetTop()-1))
				} else {
					r, err = data.Exec()
				}
				if err != nil {
					s.RaiseError(err.Error())
				}
				return Result.New(s, r)
			})
		}).
		AddMethodCast(`queryMany`, `(JSON)JSON 	 query many from json array with parameters`, func(s *lua.LState, data *sqlx.Stmt) int {
			return Raise(s, func() int {
				if s.GetTop() != 2 {
					s.RaiseError("queries(SQL,JsonArrayOfNamedParameters)")
				}
				j := json.JSON.Check(s, 2)
				if i1, ok := j.Data().([]any); ok {
					rs := Wrap(make([]any, 0, 1))
					for iy, val := range i1 {
						if m, ok := val.([]any); !ok {
							s.RaiseError("require json array of array which is not at %d", iy)
						} else {
							r, err := data.Queryx(m...)
							if err != nil {
								s.RaiseError("process %d: %s", iy, err.Error())
							}
							for r.Next() {
								m := make(map[string]any)
								if err = r.MapScan(m); err != nil {
									s.RaiseError(err.Error())
									_ = r.Close()
								}
								if err = rs.ArrayAppend(m); err != nil {
									s.RaiseError(err.Error())
									_ = r.Close()
								}
							}
							_ = r.Close()
						}
					}
					return json.JSON.New(s, rs)
				}

				s.ArgError(2, "must a json array of objects")
				return 0
			})

		}).
		AddMethodCast(`execMany`, `(JSON)number 	 exec SQL with json array of array parameters`, func(s *lua.LState, data *sqlx.Stmt) int {
			return Raise(s, func() int {
				if s.GetTop() != 2 {
					s.RaiseError("execs(SQL,JsonArrayOfNamedParameters)")
				}
				j := json.JSON.Check(s, 2)
				if i1, ok := j.Data().([]any); ok {
					n := int64(0)
					for iy, val := range i1 {
						if m, ok := val.([]any); !ok {
							s.RaiseError("require json array of array which is not at %d", iy)
						} else {
							r, err := data.Exec(m...)
							if err != nil {
								s.RaiseError("process %d: %s", iy, err.Error())
							}
							n += fn.Panic1(r.RowsAffected())
						}
					}
					s.Push(lua.LNumber(n))
					return 1
				}

				s.ArgError(3, "must a json array of objects")
				return 0
			})
		}).
		AddMethodCast(`close`, `() 	 close statement`, func(s *lua.LState, data *sqlx.Stmt) int {
			err := data.Close()
			if err != nil {
				s.RaiseError(`close database: %s`, err)
			}
			return 0
		})

	NamedStmt = NewTypeCast(func(a any) (v *sqlx.NamedStmt, ok bool) { v, ok = a.(*sqlx.NamedStmt); return }, `NamedStmt`, `sqlx.NamedStmt`, false, `none`, nil).
		AddMethodCast(`query`, `(JSON)JSON 	 param must a json Object`, func(s *lua.LState, data *sqlx.NamedStmt) int {
			return Raise(s, func() int {
				var r *sqlx.Rows
				var err error
				if s.GetTop() > 2 {
					s.ArgError(3, fmt.Sprintf("argument error with %d args", s.GetTop()-1))
				}
				j := json.JSON.Check(s, 2)
				if m, ok := j.Data().(map[string]any); ok {
					r, err = data.Queryx(m)
				} else {
					s.ArgError(2, "must a json object")
				}

				if err != nil {
					s.RaiseError("query error :%s", err)
				}
				defer r.Close()
				rs := Wrap([]any{})
				for r.Next() {
					m := make(map[string]any)
					err = r.MapScan(m)
					if err != nil {
						s.RaiseError(err.Error())
					}
					fn.Panic(rs.ArrayAppend(m))
				}
				return json.JSON.New(s, rs)
			})
		}).
		AddMethodCast(`exec`, `(JSON)Result 	 param must a json Object`, func(s *lua.LState, data *sqlx.NamedStmt) int {
			return Raise(s, func() int {
				var r sql.Result
				var err error
				if s.GetTop() > 2 {
					s.ArgError(3, fmt.Sprintf("argument error with %d args", s.GetTop()-1))
				}
				j := json.JSON.Check(s, 2)
				if m, ok := j.Data().(map[string]any); ok {
					r, err = data.Exec(m)
				} else {
					s.ArgError(2, "must a json object")
				}

				if err != nil {
					s.RaiseError(err.Error())
				}
				return Result.New(s, r)
			})
		}).
		AddMethodCast(`queryMany`, `(JSON)JSON 	 query many with json array of named parameters`, func(s *lua.LState, data *sqlx.NamedStmt) int {
			return Raise(s, func() int {
				if s.GetTop() != 2 {
					s.RaiseError("queries(SQL,JsonArrayOfNamedParameters)")
				}
				j := json.JSON.Check(s, 2)
				if i1, ok := j.Data().([]any); ok {
					rs := Wrap(make([]any, 0, 1))
					for iy, val := range i1 {
						if m, ok := val.(map[string]any); !ok {
							s.RaiseError("require json array of objects which is not at %d", iy)
						} else {
							r, err := data.Queryx(m)
							if err != nil {
								s.RaiseError("process %d: %s", iy, err.Error())
							}
							for r.Next() {
								m := make(map[string]any)
								if err = r.MapScan(m); err != nil {
									s.RaiseError(err.Error())
									_ = r.Close()
								}
								if err = rs.ArrayAppend(m); err != nil {
									s.RaiseError(err.Error())
									_ = r.Close()
								}
							}
							_ = r.Close()
						}
					}
					return json.JSON.New(s, rs)
				}

				s.ArgError(2, "must a json array of objects")
				return 0
			})

		}).
		AddMethodCast(`execMany`, `(JSON)number 	 exec with json array of named parameters`, func(s *lua.LState, data *sqlx.NamedStmt) int {
			if s.GetTop() != 2 {
				s.RaiseError("execs(SQL,JsonArrayOfNamedParameters)")
				return 0
			}
			j := json.JSON.Check(s, 2)
			if i1, ok := j.Data().([]any); ok {
				n := int64(0)
				for iy, val := range i1 {
					if m, ok := val.(map[string]any); !ok {
						s.RaiseError("require json array of objects which is not at %d", iy)
					} else {
						r, err := data.Exec(m)
						if err != nil {
							s.RaiseError("process %d: %s", iy, err.Error())
						}
						n += fn.Panic1(r.RowsAffected())
					}
				}
				s.Push(lua.LNumber(n))
				return 1
			}
			s.ArgError(3, "must a json array of objects")
			return 0
		}).
		AddMethodCast(`close`, `() 	 close statement`, func(s *lua.LState, data *sqlx.NamedStmt) int {
			err := data.Close()
			if err != nil {
				s.RaiseError(`close database: %s`, err)
			}
			return 0
		})

	fn.Panic(Register(MODULE.AddModule(DB).
		AddModule(Stmt).
		AddModule(NamedStmt).
		AddModule(Result).
		AddModule(TX)))
}
