package resources

import "testing"

func TestNewRegistryRegistersBuiltInTypes(t *testing.T) {
	r := NewRegistry()

	for _, typ := range []Type{TypeDatabase, TypeCache, TypeVolume, TypeSecret, TypeDomain} {
		if !r.Supports(typ) {
			t.Errorf("Supports(%q) = false, want true", typ)
		}
	}

	if got, want := len(r.Types()), 5; got != want {
		t.Errorf("len(Types()) = %d, want %d", got, want)
	}
}

func TestRegistrySupportsRejectsUnknownType(t *testing.T) {
	r := NewRegistry()

	if r.Supports("QUEUE") {
		t.Error("Supports(QUEUE) = true, want false")
	}
}

func TestRegistryRegisterAddsCustomType(t *testing.T) {
	r := NewRegistry()
	r.Register(TypeDescriptor{Type: "QUEUE", DisplayName: "Queue"})

	if !r.Supports("QUEUE") {
		t.Error("Supports(QUEUE) = false, want true after Register")
	}

	desc, ok := r.Get("QUEUE")
	if !ok {
		t.Fatal("Get(QUEUE) ok = false, want true")
	}
	if desc.DisplayName != "Queue" {
		t.Errorf("Get(QUEUE).DisplayName = %q, want Queue", desc.DisplayName)
	}
}

func TestRegistryRegisterReplacesExistingType(t *testing.T) {
	r := NewRegistry()
	r.Register(TypeDescriptor{Type: TypeDatabase, DisplayName: "Relational Database"})

	desc, ok := r.Get(TypeDatabase)
	if !ok {
		t.Fatal("Get(TypeDatabase) ok = false, want true")
	}
	if desc.DisplayName != "Relational Database" {
		t.Errorf("Get(TypeDatabase).DisplayName = %q, want %q", desc.DisplayName, "Relational Database")
	}
}

func TestRegistryGetReportsUnknownType(t *testing.T) {
	r := NewRegistry()

	if _, ok := r.Get("QUEUE"); ok {
		t.Error("Get(QUEUE) ok = true, want false")
	}
}
