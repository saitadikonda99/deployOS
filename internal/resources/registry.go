package resources

import "sync"

// Type identifies what kind of infrastructure a Resource represents.
// It's a plain string, not a closed Go enum, so a future resource type
// can be registered without a breaking change to this package - the
// Registry below, not the type system, is what decides which values
// are actually accepted.
type Type string

// Built-in resource types. Every other Type is possible in principle
// (Registry.Register accepts any of them) but only these five are
// registered by NewRegistry.
const (
	TypeDatabase Type = "DATABASE"
	TypeCache    Type = "CACHE"
	TypeVolume   Type = "VOLUME"
	TypeSecret   Type = "SECRET"
	TypeDomain   Type = "DOMAIN"
)

// TypeDescriptor describes one resource type known to a Registry. It
// carries no provisioning behavior - what it takes to actually create
// a DATABASE or a DOMAIN is a future Provisioner's job (see
// interfaces.go and docs/resource-engine.md's "Future provisioning
// architecture" section), not this package's.
type TypeDescriptor struct {
	Type Type
	// DisplayName is a human-readable label, e.g. for the dashboard.
	DisplayName string
}

// Registry is the set of resource types DeployOS currently knows
// about. Adding a new type - built-in or, in the future, plugin-
// provided - means registering a TypeDescriptor; Resource.Validate
// never hardcodes the five built-ins itself, it only asks a Registry.
type Registry struct {
	mu    sync.RWMutex
	types map[Type]TypeDescriptor
}

// NewRegistry builds a Registry pre-populated with DeployOS's five
// built-in resource types.
func NewRegistry() *Registry {
	r := &Registry{types: make(map[Type]TypeDescriptor, 5)}
	for _, d := range []TypeDescriptor{
		{Type: TypeDatabase, DisplayName: "Database"},
		{Type: TypeCache, DisplayName: "Cache"},
		{Type: TypeVolume, DisplayName: "Volume"},
		{Type: TypeSecret, DisplayName: "Secret"},
		{Type: TypeDomain, DisplayName: "Domain"},
	} {
		r.Register(d)
	}
	return r
}

// Register adds desc to the registry, replacing any existing
// descriptor for the same Type.
func (r *Registry) Register(desc TypeDescriptor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.types[desc.Type] = desc
}

// Supports reports whether t has been registered.
func (r *Registry) Supports(t Type) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.types[t]
	return ok
}

// Get returns the TypeDescriptor registered for t.
func (r *Registry) Get(t Type) (TypeDescriptor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	d, ok := r.types[t]
	return d, ok
}

// Types returns every registered Type, in no particular order.
func (r *Registry) Types() []Type {
	r.mu.RLock()
	defer r.mu.RUnlock()
	types := make([]Type, 0, len(r.types))
	for t := range r.types {
		types = append(types, t)
	}
	return types
}

// defaultRegistry backs the package-level NewResource/Resource.Validate
// convenience path, so most callers never need to build or pass a
// Registry themselves. It holds only the five built-in types.
var defaultRegistry = NewRegistry()
