package resources

import (
	"errors"
	"testing"
	"time"
)

func TestNewResourceSuccess(t *testing.T) {
	res, err := NewResource("user-1", "primary-db", TypeDatabase)
	if err != nil {
		t.Fatalf("NewResource() error = %v", err)
	}

	if res.ID == "" {
		t.Error("ID is empty, want a generated ID")
	}
	if res.UserID != "user-1" {
		t.Errorf("UserID = %q, want user-1", res.UserID)
	}
	if res.Name != "primary-db" {
		t.Errorf("Name = %q, want primary-db", res.Name)
	}
	if res.Type != TypeDatabase {
		t.Errorf("Type = %q, want %q", res.Type, TypeDatabase)
	}
	if res.Status != StatusPending {
		t.Errorf("Status = %q, want %q", res.Status, StatusPending)
	}
	if res.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero, want set")
	}
	if res.UpdatedAt.IsZero() {
		t.Error("UpdatedAt is zero, want set")
	}
}

func TestNewResourceSupportsAllBuiltInTypes(t *testing.T) {
	for _, typ := range []Type{TypeDatabase, TypeCache, TypeVolume, TypeSecret, TypeDomain} {
		if _, err := NewResource("user-1", "my-resource", typ); err != nil {
			t.Errorf("NewResource(type=%q) error = %v, want nil", typ, err)
		}
	}
}

func TestNewResourceValidatesName(t *testing.T) {
	tests := []struct {
		name    string
		resName string
		wantErr error
	}{
		{"empty", "", ErrEmptyName},
		{"uppercase", "MyDB", ErrInvalidName},
		{"spaces", "my db", ErrInvalidName},
		{"leading hyphen", "-my-db", ErrInvalidName},
		{"trailing hyphen", "my-db-", ErrInvalidName},
		{"underscore", "my_db", ErrInvalidName},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewResource("user-1", tt.resName, TypeDatabase)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("NewResource(name=%q) error = %v, want %v", tt.resName, err, tt.wantErr)
			}
		})
	}
}

func TestNewResourceRequiresUserID(t *testing.T) {
	_, err := NewResource("", "primary-db", TypeDatabase)
	if !errors.Is(err, ErrEmptyUserID) {
		t.Errorf("NewResource(userID=\"\") error = %v, want %v", err, ErrEmptyUserID)
	}
}

func TestNewResourceRejectsUnsupportedType(t *testing.T) {
	_, err := NewResource("user-1", "my-resource", Type("QUEUE"))
	if !errors.Is(err, ErrUnsupportedType) {
		t.Errorf("NewResource(type=QUEUE) error = %v, want %v", err, ErrUnsupportedType)
	}
}

func TestResourceValidateRejectsInvalidStatus(t *testing.T) {
	res, err := NewResource("user-1", "primary-db", TypeDatabase)
	if err != nil {
		t.Fatalf("NewResource() error = %v", err)
	}
	res.Status = Status("not-a-real-status")

	if err := res.Validate(); !errors.Is(err, ErrInvalidStatus) {
		t.Errorf("Validate() error = %v, want %v", err, ErrInvalidStatus)
	}
}

func TestResourceValidateConfig(t *testing.T) {
	res, err := NewResource("user-1", "primary-db", TypeDatabase)
	if err != nil {
		t.Fatalf("NewResource() error = %v", err)
	}
	res.Config = map[string]string{"": "value"}

	if err := res.Validate(); !errors.Is(err, ErrEmptyConfigKey) {
		t.Errorf("Validate() error = %v, want %v", err, ErrEmptyConfigKey)
	}
}

func TestResourceValidateAcceptsWellFormedConfig(t *testing.T) {
	res, err := NewResource("user-1", "primary-db", TypeDatabase)
	if err != nil {
		t.Fatalf("NewResource() error = %v", err)
	}
	res.Config = map[string]string{"engine": "postgres", "version": "16"}

	if err := res.Validate(); err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

func TestResourceValidateWithRegistryUsesGivenRegistry(t *testing.T) {
	custom := NewRegistry()
	custom.Register(TypeDescriptor{Type: "QUEUE", DisplayName: "Queue"})

	res := Resource{
		ID:        ID("res-1"),
		UserID:    "user-1",
		Name:      "my-queue",
		Type:      "QUEUE",
		Status:    StatusPending,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	// The default registry doesn't know QUEUE...
	if err := res.Validate(); !errors.Is(err, ErrUnsupportedType) {
		t.Errorf("Validate() error = %v, want %v", err, ErrUnsupportedType)
	}
	// ...but a custom registry that registers it does.
	if err := res.ValidateWithRegistry(custom); err != nil {
		t.Errorf("ValidateWithRegistry(custom) error = %v, want nil", err)
	}
}

func TestIDValidate(t *testing.T) {
	if err := ID("res-1").Validate(); err != nil {
		t.Errorf("ID(\"res-1\").Validate() error = %v, want nil", err)
	}
	if err := ID("").Validate(); !errors.Is(err, ErrEmptyID) {
		t.Errorf("ID(\"\").Validate() error = %v, want %v", err, ErrEmptyID)
	}
}
