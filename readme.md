# GLU (glucose) - [gopher-lua](https://github.com/yuin/gopher-lua) module extensions

## Packages

1. √ `glu` the core module:
   1. Define helper `Module` and `Type` for easier register user library and user type;
   2. Define global LState pool for reuse;
   3. Define Registry for `Modulars`, with optional auto-injection;
   4. Support `Help(string?)` for help information;
2. √ `json` dynamic json library base on [Jeffail/gabs](https://github.com/Jeffail/gabs/v2)
3. √ `http` http server and client library base on [gorilla/mux](https://github.com/gorilla/mux), depends on `json`

## Samples

1. print help
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
2. http server
   ```lua
   local http=require('http')
   local server=http.Server.new(':8081') --new Server with listen address
   server:get('/',[[ -- the handler is string lua script
               local c=... --only parameter is http.Ctx
               c:sendJson(c:query('p')) --query should legal JSON string
           ]])
   server:start(false)
   while (true) do	end
   ```
3. http client
   ```lua
    local res,err=require('http').Client.new(5):get('http://github.com')
    local txt=res:body()
    print(err)
    print(txt)
   ```

## Support this project

1. offer your ideas

2. fork and pull

## License

MIT as gopher-lua did
