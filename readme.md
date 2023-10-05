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

1. usage

```go
package sample

import (
	"fmt"
	"github.com/ZenLiuCN/glu/v3"
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
	return float64(vm.CheckNumber(1))
}
```

2. print help
```lua
   local http=require('http')
   local json=require('json')
   print(json.help()) --will print comma split keyword list
   print(http.help('?')) --will print module help
   print(http.Server.help('?')) --will print type constructor help
   for word in string.gmatch(http.Server.help(), '([^,]+)') do
      print(http.Server.help(word)) --will print method constructor help
   end
   print(http.CTX.help('?'))
   for word in string.gmatch(http.CTX.help(), '([^,]+)') do
      print(http.CTX.Help(word))
   end
```

3. http server
```lua
local http=require('http')
local server=http.Server.new(':8081') --new Server with listen address
function handle(c)
 c:sendString(c:query('p'))
end
server:get('/',handle)
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

## Changes `v2.x`

Those are record start at version `2.0.2`

1. `v2.0.2` :
    + add module `sqlx` with `sqlx.DB`,`sqlx.Result`
    + add function `of(jsonString):Json` in module `json`
2. `v2.0.3` :
    + adding `sqlx.Tx`,`sqlx.Stmt`,`sqlx.NamedStmt` to module `sqlx`
3. `v2.0.4` :
    + add `Json:get(string|number)` to module `json`, which will replace `Json:at`
    + add `x:execMany` and `x:queryMany` to module `sqlx`
    + add `sqlx.encB64` and `sqlx.decB64` to module `sqlx`
      for [numeric issue](https://github.com/jmoiron/sqlx/issues/289)
        + when use [pgx](https://github.com/jackc/pgx/) for postgresql there will no such needs.
4. `v2.0.5` :
    + add `sqlx:to_num` and `sqlx:from_num` to module `sqlx`, which convert json array of objects numeric fields from|to
      binary
   
## Changes `v3.x`

Those are record start at version `3.0.1`
1. `v3.0.1`:
    + simplify core api
      + `Get`: get a pooled VM
      + `Put`: return a pooled VM
      + `Register`: register a modular
      + `NewModule`: create a Module
      + `NewSimpleType`: create a Simple Type
      + `NewType`: create a Type
      + `NewTypeCast`: create a Type with auto cast
      + `modular.AddFunc`: add a function to Module or Type
      + `modular.AddField`: add a field to Module or Type
      + `modular.AddFieldSupplier`: add a field by supplier to Module or Type
      + `modular.AddModule`: add a sub-module
      + `Type.New`: create new instance and push to stack
      + `Type.NewValue`: create new instance only
      + `Type.Cast`: check if is auto cast Type
      + `Type.Check`: check stack value is Type, must a Cast Type
      + `Type.CheckSelf`: check top stack value is Type, must a Cast Type
      + `Type.CheckUserData`: check user-data value is Type, must a Cast Type
      + `Type.Caster`: the caster or nil
      + `Type.AddMethod`: add a Method
      + `Type.AddMethodUserData`: add a Method use UserData as receiver
      + `Type.AddMethodCast`: add a Method use specific Type as receiver,  must a Cast Type
      + `Type.Override`: override a meta function
      + `Type.OverrideUserData`: override a meta function  use UserData as receiver
      + `Type.OverrideCast`: override a meta function use specific Type as receiver,  must a Cast Type
    + `json`
      + `json.stringify`: convert JSON to json string
      + `json.parse`: create JSON from json string
      + `json.of`: create JSON from lua value
      + `JSON.new`: create new JSON from a json string
      + `JSON:json`: convert JSON to json string
      + `JSON:path`: fetch JSON element at path
      + `JSON:exists`: check JSON path exists
      + `JSON:get`: fetch JSON element at path or index of array
      + `JSON:set`: set JSON element at path or index of array, nil value will delete.
      + `JSON:type`: get JSON element type at path.
      + `JSON:append`: append JSON element at path, returns error message.
      + `JSON:isArray`: check if JSON element at path is JSON array.
      + `JSON:isObject`: check if JSON element at path is JSON Object.
      + `JSON:bool`: fetch JSON element at path, which should be a boolean.
      + `JSON:string`: fetch JSON element at path, which should be a string.
      + `JSON:number`: fetch JSON element at path, which should be a number.
      + `JSON:size`: fetch JSON size at path,if not array or object, returns nil.
      + `JSON:raw`: fetch JSON element at path, and convert to lua value.
      + `tostring(JSON)`: convert JSON to json string.
   + `http`
     + `http.Server`: the http server
     + `http.Client`: the http client
     + `http.CTX`: the http request context
     + `http.Response`: the http response
     + `CTX:vars`: path variables by name
     + `CTX:header`: request header by name
     + `CTX:query`: request query by name
     + `CTX:method`: request method
     + `CTX:body`: request body as JSON
     + `CTX:setHeader`: set response header
     + `CTX:status`: set response status
     + `CTX:sendJson`: send JSON as response body and end process
     + `CTX:sendString`: send string as response body and end process
     + `CTX:sendFile`: send File as response body and end process
     + `Server.new`: create new http.Server listen at address
     + `Server:stop`: shutdown http server
     + `Server:running`: check if server is running
     + `Server:start`: start server to listen
     + `Server:route`: declare route without limit request method
     + `Server:get`: declare route for GET method
     + `Server:post`: declare route for POST method
     + `Server:put`: declare route for PUT method
     + `Server:head`: declare route for HEAD method
     + `Server:patch`: declare route for PATCH method
     + `Server:delete`: declare route for DELETE method
     + `Server:connect`: declare route for CONNECT method
     + `Server:options`: declare route for OPTIONS method
     + `Server:trace`: declare route for TRACE method
     + `Server:files`: declare route for serve with files
     + `Server:release`: free server resources
     + `Server.pool`: server pool size
     + `Server.poolKeys`: server pool keys
     + `Server.pooled`: fetch server from pool by key
     + `Response:statusCode`: response status code
     + `Response:status`: response status text
     + `Response:size`: response content size
     + `Response:header`: response headers
     + `Response:body`: response body as string
     + `Response:bodyJson`: response body as JSON
     + `Client.new`: create new http client
     + `Client:get`: send GET request
     + `Client:post`: send POST request
     + `Client:head`: send HEAD request
     + `Client:form`: send POST request with form
     + `Client:request`: send request with string data
     + `Client:requestJson`: send request with JSON data
     + `Client:release`:  free client resources
     + `Client.pool`:  client pool size
     + `Client.poolKeys`:  client pool keys
     + `Client.pooled`:  get client from pool by key
   + `sqlx`
     + `sqlx.DB`: sqlx database
     + `sqlx.Tx`: sqlx transaction 
     + `sqlx.Stmt`: sqlx prepared statement 
     + `sqlx.NamedStmt`: sqlx prepared named statement 
     + `sqlx.Result`: sql execute result
     + `sqlx.connect`: connect to database
     + `sqlx.encB64`: encode string to base64
     + `sqlx.decB64`: decode base64 to string
     + `sqlx.from_num`: encode string of decimal to base64
     + `sqlx.to_num`: decode base64 to string of decimal
     + `DB.new`: connect to database
     + `DB:query`: query SQL data as JSON
     + `DB:exec`: execute SQL fetch Result
     + `DB:queryMany`: query SQL with batch of parameters
     + `DB:execMany`: execute SQL with batch of parameters
     + `DB:begin`: begin transaction
     + `DB:prepare`: prepare statement
     + `DB:prepareNamed`: prepare named statement
     + `DB:close`: close database
     + `Tx:query`: query SQL data as JSON
     + `Tx:exec`: execute SQL fetch Result
     + `Tx:queryMany`: query SQL with batch of parameters
     + `Tx:execMany`: execute SQL with batch of parameters
     + `Tx:prepare`: prepare statement
     + `Tx:prepareNamed`: prepare named statement
     + `Tx:commit`:commit transaction
     + `Tx:rollback`:rollback transaction
     + `Stmt:query`: query data as JSON
     + `Stmt:exec`: execute fetch Result
     + `Stmt:queryMany`: query with batch of parameters
     + `Stmt:execMany`: execute with batch of parameters
     + `Stmt:close`: close statement
     + `NamedStmt:query`: query data as JSON
     + `NamedStmt:exec`: execute fetch Result
     + `NamedStmt:queryMany`: query with batch of parameters
     + `NamedStmt:execMany`: execute with batch of parameters
     + `NamedStmt:close`: close statement
     + `Result:lastID`: last inserted ID
     + `Result:rows`: affected rows