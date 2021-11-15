# GLU (glucose) - [gopher-lua](https://github.com/yuin/gopher-lua) module extensions

## Packages

1. √ `glu` the core module:
   1. Define helper `Module` and `Type` for easier register user library and user type;
   2. Define global LState pool for reuse;
   3. Define Registry for `Modulars`, with optional auto-injection;
   4. Support `Help(string?)` for help information;
2. √ `json` dynamic json library base on [Jeffail/gabs](https://github.com/Jeffail/gabs/v2)
3. √ `http` http server and client library base on [gorilla/mux](https://github.com/gorilla/mux), depends on `json`

## License

MIT as gopher-lua did
