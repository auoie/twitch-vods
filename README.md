# README

There is a scraper in `./cmd/testingScraper` that always runs and uploads to a Postgres database.
The database schema is in `./sqlc/migrations`.
There is an API layer in `./cmd/stringApi` that reads from the database.
