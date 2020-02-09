# go-migrate

A Go library for handling database migrations.

## Features

* **Safe to run on application startup**: The migration versions table is locked, preventing 
multiple copies of the same migrations from executing at the same time.
* **Migrations are code**: Meaning you don't have to worry about how to package up `.sql` files, 
etc. into your Go binary, or transporting the `.sql` files with your application.
* **Simple versioning**: Versions are just numbers - easy to sort (timestamps make good versions).
* **Configurable versions table**: Migration drivers have relevant configuration exposed.
* **Designed to be integrated in to your code**: Output is handled by implementing an `EventHandler`
where you can use your own logger, etc.
* **Namespaced migrations**: If you have multiple databases to migrate in one app, you can keep the
migrations completely separate, and run them separately too.
