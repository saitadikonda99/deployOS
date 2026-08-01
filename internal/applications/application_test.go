package applications

import (
	"errors"
	"testing"

	"github.com/saitadikonda99/deployOS/internal/resources"
)

func TestNewApplicationSuccess(t *testing.T) {
	app, err := NewApplication("user-1", "my-app", RuntimeDocker)
	if err != nil {
		t.Fatalf("NewApplication() error = %v", err)
	}

	if app.ID == "" {
		t.Error("ID is empty, want a generated ID")
	}
	if app.UserID != "user-1" {
		t.Errorf("UserID = %q, want user-1", app.UserID)
	}
	if app.Name != "my-app" {
		t.Errorf("Name = %q, want my-app", app.Name)
	}
	if app.Runtime != RuntimeDocker {
		t.Errorf("Runtime = %q, want %q", app.Runtime, RuntimeDocker)
	}
	if app.Status != StatusCreated {
		t.Errorf("Status = %q, want %q", app.Status, StatusCreated)
	}
	if app.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero, want set")
	}
	if app.UpdatedAt.IsZero() {
		t.Error("UpdatedAt is zero, want set")
	}
	if app.Repository != nil {
		t.Errorf("Repository = %+v, want nil (optional, unset by default)", app.Repository)
	}
}

func TestNewApplicationValidatesName(t *testing.T) {
	tests := []struct {
		name    string
		appName string
		wantErr error
	}{
		{"empty", "", ErrEmptyName},
		{"uppercase", "MyApp", ErrInvalidName},
		{"spaces", "my app", ErrInvalidName},
		{"leading hyphen", "-my-app", ErrInvalidName},
		{"trailing hyphen", "my-app-", ErrInvalidName},
		{"underscore", "my_app", ErrInvalidName},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewApplication("user-1", tt.appName, RuntimeDocker)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("NewApplication(name=%q) error = %v, want %v", tt.appName, err, tt.wantErr)
			}
		})
	}
}

func TestNewApplicationAcceptsValidNames(t *testing.T) {
	for _, name := range []string{"a", "my-app", "app123", "my-app-2"} {
		if _, err := NewApplication("user-1", name, RuntimeDocker); err != nil {
			t.Errorf("NewApplication(name=%q) error = %v, want nil", name, err)
		}
	}
}

func TestNewApplicationRequiresUserID(t *testing.T) {
	_, err := NewApplication("", "my-app", RuntimeDocker)
	if !errors.Is(err, ErrEmptyUserID) {
		t.Errorf("NewApplication(userID=\"\") error = %v, want %v", err, ErrEmptyUserID)
	}
}

func TestNewApplicationRequiresRuntime(t *testing.T) {
	_, err := NewApplication("user-1", "my-app", "")
	if !errors.Is(err, ErrEmptyRuntime) {
		t.Errorf("NewApplication(runtime=\"\") error = %v, want %v", err, ErrEmptyRuntime)
	}
}

func TestApplicationValidateRejectsInvalidStatus(t *testing.T) {
	app, err := NewApplication("user-1", "my-app", RuntimeDocker)
	if err != nil {
		t.Fatalf("NewApplication() error = %v", err)
	}
	app.Status = Status("not-a-real-status")

	if err := app.Validate(); !errors.Is(err, ErrInvalidStatus) {
		t.Errorf("Validate() error = %v, want %v", err, ErrInvalidStatus)
	}
}

func TestApplicationValidateEnvironmentVariables(t *testing.T) {
	app, err := NewApplication("user-1", "my-app", RuntimeDocker)
	if err != nil {
		t.Fatalf("NewApplication() error = %v", err)
	}
	app.EnvironmentVariables = map[string]string{"": "value"}

	if err := app.Validate(); !errors.Is(err, ErrEmptyEnvVarName) {
		t.Errorf("Validate() error = %v, want %v", err, ErrEmptyEnvVarName)
	}
}

func TestApplicationValidateSecretRefs(t *testing.T) {
	tests := []struct {
		name string
		ref  SecretRef
	}{
		{"missing env var", SecretRef{SecretName: "db-password"}},
		{"missing secret name", SecretRef{EnvVar: "DB_PASSWORD"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, err := NewApplication("user-1", "my-app", RuntimeDocker)
			if err != nil {
				t.Fatalf("NewApplication() error = %v", err)
			}
			app.SecretRefs = []SecretRef{tt.ref}

			if err := app.Validate(); !errors.Is(err, ErrInvalidSecretRef) {
				t.Errorf("Validate() error = %v, want %v", err, ErrInvalidSecretRef)
			}
		})
	}
}

func TestApplicationValidateSecretRefsAccepted(t *testing.T) {
	app, err := NewApplication("user-1", "my-app", RuntimeDocker)
	if err != nil {
		t.Fatalf("NewApplication() error = %v", err)
	}
	app.SecretRefs = []SecretRef{{EnvVar: "DB_PASSWORD", SecretName: "db-password"}}

	if err := app.Validate(); err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

func TestApplicationValidateVolumes(t *testing.T) {
	tests := []struct {
		name   string
		volume Volume
	}{
		{"missing host path", Volume{ContainerPath: "/data"}},
		{"missing container path", Volume{HostPath: "/srv/data"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, err := NewApplication("user-1", "my-app", RuntimeDocker)
			if err != nil {
				t.Fatalf("NewApplication() error = %v", err)
			}
			app.Volumes = []Volume{tt.volume}

			if err := app.Validate(); !errors.Is(err, ErrInvalidVolume) {
				t.Errorf("Validate() error = %v, want %v", err, ErrInvalidVolume)
			}
		})
	}
}

func TestApplicationValidateDomains(t *testing.T) {
	app, err := NewApplication("user-1", "my-app", RuntimeDocker)
	if err != nil {
		t.Fatalf("NewApplication() error = %v", err)
	}
	app.Domains = []string{""}

	if err := app.Validate(); !errors.Is(err, ErrInvalidDomain) {
		t.Errorf("Validate() error = %v, want %v", err, ErrInvalidDomain)
	}
}

func TestApplicationValidateAcceptsOptionalFieldsUnset(t *testing.T) {
	app, err := NewApplication("user-1", "my-app", RuntimeDocker)
	if err != nil {
		t.Fatalf("NewApplication() error = %v", err)
	}

	// Repository, EnvironmentVariables, SecretRefs, Volumes, and
	// Domains are all optional - a freshly created Application with
	// none of them set must still validate.
	if err := app.Validate(); err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

func TestApplicationValidateAcceptsConfiguredRepository(t *testing.T) {
	app, err := NewApplication("user-1", "my-app", RuntimeDocker)
	if err != nil {
		t.Fatalf("NewApplication() error = %v", err)
	}
	app.Repository = &SourceRepository{URL: "https://github.com/example/my-app", Branch: "main"}

	if err := app.Validate(); err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

func TestIDValidate(t *testing.T) {
	if err := ID("app-1").Validate(); err != nil {
		t.Errorf("ID(\"app-1\").Validate() error = %v, want nil", err)
	}
	if err := ID("").Validate(); !errors.Is(err, ErrEmptyID) {
		t.Errorf("ID(\"\").Validate() error = %v, want %v", err, ErrEmptyID)
	}
}

func TestRuntimeValidate(t *testing.T) {
	if err := RuntimeDocker.Validate(); err != nil {
		t.Errorf("RuntimeDocker.Validate() error = %v, want nil", err)
	}
	if err := Runtime("").Validate(); !errors.Is(err, ErrEmptyRuntime) {
		t.Errorf("Runtime(\"\").Validate() error = %v, want %v", err, ErrEmptyRuntime)
	}
}

// TestApplicationResourceRefsRelationship exercises the actual
// relationship between the two domain models: an Application can
// reference a Resource (internal/resources) that was constructed and
// validated entirely independently of any Application. This is what
// "Applications reference Resources by ID" means in practice - the
// two packages cooperate through nothing more than resources.ID.
func TestApplicationResourceRefsRelationship(t *testing.T) {
	db, err := resources.NewResource("user-1", "primary-db", resources.TypeDatabase)
	if err != nil {
		t.Fatalf("resources.NewResource() error = %v", err)
	}
	cache, err := resources.NewResource("user-1", "session-cache", resources.TypeCache)
	if err != nil {
		t.Fatalf("resources.NewResource() error = %v", err)
	}

	app, err := NewApplication("user-1", "my-app", RuntimeDocker)
	if err != nil {
		t.Fatalf("NewApplication() error = %v", err)
	}
	app.ResourceRefs = []resources.ID{db.ID, cache.ID}

	if err := app.Validate(); err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
	if len(app.ResourceRefs) != 2 {
		t.Errorf("len(ResourceRefs) = %d, want 2", len(app.ResourceRefs))
	}
}

func TestApplicationValidateAcceptsNoResourceRefs(t *testing.T) {
	app, err := NewApplication("user-1", "my-app", RuntimeDocker)
	if err != nil {
		t.Fatalf("NewApplication() error = %v", err)
	}

	// ResourceRefs is optional - an Application with none set (the
	// common case until it depends on any infrastructure) must still
	// validate.
	if err := app.Validate(); err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

func TestApplicationValidateRejectsEmptyResourceRef(t *testing.T) {
	app, err := NewApplication("user-1", "my-app", RuntimeDocker)
	if err != nil {
		t.Fatalf("NewApplication() error = %v", err)
	}
	app.ResourceRefs = []resources.ID{""}

	if err := app.Validate(); !errors.Is(err, ErrInvalidResourceRef) {
		t.Errorf("Validate() error = %v, want %v", err, ErrInvalidResourceRef)
	}
}
