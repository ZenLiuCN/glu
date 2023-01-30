package glu

func Example() {
	// fetch an instance
	vm := Get()
	vm.OpenLibs()
	err := vm.DoString(`print('hello lua')`)
	if err != nil {
		return 
	}
}
