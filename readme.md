# GLU (glucose) - [gopher-lua](https://github.com/yuin/gopher-lua) module extensions

## Summary

### requires

+ go 1.17 (as gopher-lua required): `under branch g17 with uri "github.com/ZenLiuCN/glu"`
+ go 1.18 (with Generic): `current master with uri "github.com/ZenLiuCN/glu/v2"`

## Packages

1. √ `glu` the core module:
    1. Define helper `Module` and `Type` for easier register user library and user type;
    2. Define global LState pool for reuse;
    3. Define Registry for `Modulars`, with optional auto-injection;
    4. Support `Help(string?)` for help information;
2. √ `json` dynamic json library base on [Jeffail/gabs](https://github.com/Jeffail/gabs/v2)
3. √ `http` http server and client library base on [gorilla/mux](https://github.com/gorilla/mux), depends on `json`
4. √ `sqlx` sqlx base on [jmoiron/sqlx](https://github.com/jmoiron/sqlx), depends on `json`, new in version `v2.0.2`

## Samples

1. use

```go
package sample

import (
	"fmt"
	"github.com/ZenLiuCN/glu/v2"
	lua "github.com/yuin/gopher-lua"
)

func main() {
	fmt.Println(DoSomeScript("1+2") == 3.0)
}
func DoSomeScript(script string) float64 {
	vm := glu.Get()
	defer glu.Put(vm)
	if err := vm.DoString(script); err != nil {
		panic(err)
	}
	return float64(vm.Pop().(lua.LNumber))
}
```

2. print help
   ```lua
      local http=require('http')
      local json=require('json')
      print(json.Help()) --will print comma split keyword list
      print(http.Help('?')) --will print module help
      print(http.Server.Help('?')) --will print type constructor help
      for word in string.gmatch(http.Server.Help(), '([^,]+)') do
         print(http.Server.Help(word)) --will print method constructor help
      end
      print(http.Ctx.Help('?'))
      for word in string.gmatch(http.Ctx.Help(), '([^,]+)') do
         print(http.Ctx.Help(word))
      end
   ```
3. http server
   ```lua
   local http=require('http')
   local server=http.Server.new(':8081') --new Server with listen address
   server:get('/',chunk([[                -- the handler is string lua script
               local c=...                --only parameter is http.Ctx
               c:sendString(c:query('p')) --query should legal JSON string
           ]]))
   server:start(false)
   while (true) do	end
   ```
4. http client
   ```lua
    local res,err=require('http').Client.new(5):get('http://github.com')
    print(err)
    if res:size()>0 then
    local txt=res:body()  
    print(txt)
    end 
   ```

## Support this project

1. offer your ideas

2. fork and pull

## License

MIT as gopher-lua did

## Changes

Those are record start at version `2.0.2`

1. `v2.0.2` :
    + add module `sqlx` with `sqlx.DB`,`sqlx.Result`
    + add function `of(jsonString):Json` in module `json`
2. `v2.0.3` :
   + adding `sqlx.Tx`,`sqlx.Stmt`,`sqlx.NamedStmt` to module `sqlx`