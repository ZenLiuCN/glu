package glu

var (
	modulars    []Modular
	moduleNames map[string]struct{}
	// holder is placeholder for a map set
	holder = struct{}{}
)

// Register modular into registry
func Register(m ...Modular) (err error) {
	if moduleNames == nil {
		moduleNames = make(map[string]struct{}, 8)
	}
	for _, mod := range m {
		if v, ok := moduleNames[mod.GetName()]; ok && v == holder {
			return ErrAlreadyExists
		}
		modulars = append(modulars, mod)
		moduleNames[mod.GetName()] = holder
	}
	return
}
