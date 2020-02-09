# go-migrate

A Go library for handling database migrations.

## Notes

* Migrations must be code, so you don't have to worry about dragging `.sql` files around when 
you're doing things like building Docker images. ✅
    * Code should be able to be generated - ideally this library should provide the template or a 
    function for generating migration code.
* Migrations must have a version that is sortable, but not be consecutive, otherwise working with
migrations in a team setting is more complex. ✅
    * Current unix timestamp sounds ideal?
* Migrations must be configurable. ✅
    * Table name (and schema, if relevant)
    * The rest would come from the connection passed in
* Migrations must create the migration versions table automatically if it's not present. ✅
* Migrations should be run in a transaction to achieve locking. ✅
    * This could be configurable. Maybe you want to run a migration in a job. Must Tx initially.
* Migrations should receive a `*sql.Tx`, and just be able to do what they want with it. ✅
* Migrations should only allow you to migrate up, not down. ✅
* Migrations could be registered by anonymously importing a package in consuming application. ✅
    * i.e. `_ "github.com/seeruk/inbox/migrations"`
* Migrations could also be registered manually. ✅
    * This would allow curried migrations, if a migration needs some other service.
* Migrations could be namespaced. ✅
    * Sometimes an application may interact with different databases, possibly even in different
    database servers. We'd need to track versions separately for each.
