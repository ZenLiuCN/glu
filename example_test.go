package glu

func Example() {
	// fetch an instance
	vm := Get()
	err := vm.DoString(`print('hello lua')`)
	if err != nil {
		return
	}
}
