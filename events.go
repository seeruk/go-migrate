package migrate

// EventHandler is a type used to allow consumers of this library to handle output themselves for
// certain events as they happen during the migration process.
type EventHandler interface {
	BeforeVersionsMigrate(versions []int)
	BeforeVersionMigrate(version int)
	AfterVersionsMigrate(versions []int)
	AfterVersionMigrate(version int)
	OnVersionSkipped(version int)
	OnVersionTableNotExists()
	OnVersionTableCreated()
	OnRollbackError(err error)
}

// NoopEventHandler is a no-op EventHandler implementation.
type NoopEventHandler struct{}

// BeforeVersionsMigrate is a no-op BeforeVersionsMigrate method.
func (n NoopEventHandler) BeforeVersionsMigrate(versions []int) {}

// BeforeVersionMigrate is a no-op BeforeVersionMigrate method.
func (n NoopEventHandler) BeforeVersionMigrate(version int) {}

// AfterVersionsMigrate is a no-op AfterVersionsMigrate method.
func (n NoopEventHandler) AfterVersionsMigrate(versions []int) {}

// AfterVersionMigrate is a no-op AfterVersionMigrate method.
func (n NoopEventHandler) AfterVersionMigrate(version int) {}

// OnVersionSkipped is a no-op OnVersionSkipped method.
func (n NoopEventHandler) OnVersionSkipped(version int) {}

// OnVersionTableNotExists is a no-op OnVersionTableNotExists method.
func (n NoopEventHandler) OnVersionTableNotExists() {}

// OnVersionTableCreated is a no-op OnVersionTableCreated method.
func (n NoopEventHandler) OnVersionTableCreated() {}

// OnRollbackError is a no-op OnRollbackError method.
func (n NoopEventHandler) OnRollbackError(err error) {}
