package glu

var (
	registry []Modular
	names    map[string]struct{}
	// ExistNode is placeholder for a map set
	ExistNode = struct{}{}
)

// Register modular into registry
func Register(m ...Modular) (err error) {
	if names == nil {
		names = make(map[string]struct{}, 8)
	}
	for _, mod := range m {
		if v, ok := names[mod.GetName()]; ok && v == ExistNode {
			return ErrAlreadyExists
		}
		registry = append(registry, mod)
		names[mod.GetName()] = ExistNode
	}
	return
}
