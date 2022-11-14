# Notes

## Reuse HTTP Connections

I issue `60 * num_of_domains` HTTP requests in the case of the StreamsCharts data.
This is wasteful.
It takes like 10 seconds.
I should probably reuse HTTP connections.
See [here](https://husni.dev/how-to-reuse-http-connection-in-go/).
You can see what's going on with

```bash
time ./govods sc-manual-get-m3u8 --time "02-10-2022 01:31" --streamer goonergooch --videoid 47238989357 --write
netstat -atn
```

See [here](https://stackoverflow.com/questions/17948827/reusing-http-connections-in-go) for discussions and mistakes people made.

## HTTP/2 Multiplexing

It seems like there is something called HTTP/2 multiplexing.
You can issue multiple requests at the time time on a single TCP connection.
See [here](https://groups.google.com/g/golang-nuts/c/5T5aiDRl_cw).
They used Wireshark to assess this.

## Goroutines with Methods

See [here](https://stackoverflow.com/questions/36121984/how-to-use-a-method-as-a-goroutine-function).

## HTTP Transport

See [here](https://www.loginradius.com/blog/engineering/tune-the-go-http-client-for-high-performance/#problem2-default-http-transport)
for remarks on the Go default transport.

## GraphQL

[This repository](https://github.com/SuperSonicHub1/twitch-graphql-api) has the schema for the
Twitch GraphQL API.

```
curl -OL https://raw.githubusercontent.com/SuperSonicHub1/twitch-graphql-api/master/schema.graphql
```

See some basic usage, see [here](https://github.com/mauricew/twitch-graphql-api/blob/master/USAGE.md).

I'm not sure how to write GraphQL queries.
To get autocompletion, I installed [GraphQL: Language Feature Support in VSCode](https://marketplace.visualstudio.com/items?itemName=GraphQL.vscode-graphql).
It takes a while for the language server to process the schema, so you'll have to wait a bit if you reload VSCode.
If it's not working, look at the LSP logs in the VSCode output.
In order for it to work, I need a file `.graphqlrc.yml`.

```yaml
schema: "schema.graphql"
documents: "twitchgql/*.graphql"
```

Then we can use [Khan/genqlient](https://github.com/Khan/genqlient) to generate the GraphQL client code.
Following the pattern from [99designs/gqlgen](https://github.com/99designs/gqlgen#quick-start),
we create a file `tools.go` that contains `Khan/genqlient` as a dependency.

```bash
go mod tidy # Khan/genqlient stays because it is in tools.go
go run github.com/Khan/genqlient
```

Note order to see how to add a request header to every request for a client, see the `genqlient` [example](https://github.com/Khan/genqlient/blob/main/example/main.go).
There are more options discussed in [this StackOverflow post](https://stackoverflow.com/questions/54088660/add-headers-for-each-http-request-using-client).

Because of the annotation

```go
//go:generate go run github.com/Khan/genqlient genqlient.yaml
```

it is possible to just use `go generate ./...` to generate the client code.
See [here](https://eli.thegreenplace.net/2021/a-comprehensive-guide-to-go-generate/) for further details on code generation in Go.

## Postgres

I'm using Prisma JS to generate automatically generate the SQL table code and
apply it to the SQL database

```bash
docker run -d --restart always\
  --name sensitive_data \
  -e POSTGRES_USER=govods \
  -e POSTGRES_PASSWORD=password \
  -e POSTGRES_DB=twitch \
  -p 5432:5432 \
  postgres
npx prisma migrate dev
pgcli postgresql://govods:password@localhost:5432/twitch
```

I use the `sqls.yml` VSCode extension for autocompletion.
My file `.vscode/sqls.yml` looks like

```yaml
lowercaseKeywords: false
connections:
  - alias: govods_project
    driver: postgresql
    proto: tcp
    user: govods
    passwd: password
    host: localhost
    port: 5432
    dbName: twitch
```

and my file `.vscode/settings.json` looks like

```json
{
  "sqls.languageServerFlags": ["-config", "./.vscode/sqls.yml"]
}
```

I use `sqlc` to generate Go code to run SQL queries.

```bash
go get -u github.com/kyleconroy/sqlc/cmd/sqlc@latest
# then add it to ./tools/tools.go
go run github.com/kyleconroy/sqlc/cmd/sqlc --help
go get -u github.com/jackc/pgx/v5 # this is required to :copyfrom
```

In order to perform dynamic-sized `IN` queries,
see [here](https://github.com/kyleconroy/sqlc/issues/167), [here](https://github.com/kyleconroy/sqlc/issues/216), [here](https://github.com/kyleconroy/sqlc/issues/218).
Basically, you need to use the `:copyfrom` annotation.

`sqlc` doesn't really support updating or upserting many rows in a single query.

```sql
-- name: UpdateManyStreams :copyfrom
UPDATE
  streams
SET
  last_updated_at = $1, max_views = $2
WHERE
  stream_id = $1;

-- name: UpsertManyStreams :copyfrom
INSERT INTO
  streams (last_updated_at, max_views, start_time, streamer_id, stream_id, streamer_login_at_start)
VALUES
  ($1, $2, $3, $4, $5, $6)
ON CONFLICT
  (stream_id)
DO
  UPDATE SET
    last_updated_at = EXCLUDED.last_updated_at,
    max_views = GREATEST(max_views, EXCLUDED.max_views);
```

I got the errors

```text
# package sqlvods
sql/queries.sql:50:1: :copyfrom requires an INSERT INTO statement
sql/queries.sql:58:1: :copyfrom is not compatible with ON CONFLICT
exit status 1
```

The batching approach between `pggen` and `sqlc` are different.
I prefer the `pggen` approach.
It allows you to combine different types of queries in a single batch.

To generate the SQL code, it should be sufficient to run

```bash
sqlc generate
```

This is buggy for batches.
In particular, it doesn't get the correct imports of `time` and `uuid`.
I'm using version 1.15.
Instead, use

```bash
go install github.com/kyleconroy/sqlc/cmd/sqlc@main
~/go/bin/sqlc generate
```

Now it should work.

I prefer the approach of using batching to using `ANY` for selecting my rows in one query
because it returns a nil pointer in the case that a row returns nothing.
When using `ANY`, it only returns the rows that were found, making it harder to figure out which SELECTs were successful.
Alternatively, I could go with the approach of creating an inner table by unnesting an array if input IDs and making a `RIGHT JOIN`
on that table of IDs with the original table.

I don't like how `:one` returns an error if the element not found.
If the database fails, I get an error.
If the `ID` is not found I get an error.
It's not possible to distinguish these two cases.
Instead, I should just use `:many` and then check the size.

The most type safe approach is probably batching with `:batchmany`, while also checking that the length
of each returned value is 1.

Running the query

```sql
SELECT s.*, r.bytes_found, r.fetched_at, r.bytes_found, length(r.gzipped_bytes) AS gzipped_bytes_length FROM streams s JOIN recordings r ON s.stream_id = r.stream_id ORDER BY r.fetched_at DESC LIMIT 40;
```

gives the error

```text
could not resize shared memory segment "/PostgreSQL.1928016196" to 33554432 bytes: No space left on device
CONTEXT:  parallel worker
```

Running the query

```sql
SELECT s.*, r.bytes_found, r.fetched_at, r.bytes_found FROM streams s JOIN recordings r ON s.stream_id = r.stream_id ORDER BY r.fetched_at DESC LIMIT 40;
```

gives results.
I'm guessing it failed because it might keep all of the gzipped bytes in memory.

## Twitch

The Twitch GraphQL resolver for videos (in particular, past broadcasts) went down for a short period.
I should not trust the graphql API to work all the time.

## TODO

- Make a way to evict the gzipped bytes every 60 days. Decide whether I should keep the recording or not.
- Return VOD to VODs list if it is still live using the GraphQL client to check.
  Alternatively, when a live vod is fetched, check if it's in the old vod queue. If it's there, remove it from the old vod queue.
- Old vods ordered by views is way too big after I stop and restart the program.

  I'm getting this.

  ```text
  2022/11/13 20:44:38 oldest time allowed: 2022-11-13 20:29:38.903041739 -0800 PST m=-899.400413453
  2022/11/13 20:44:38 stalestVod: 2022-11-13 20:29:48.939 +0000 UTC
  ```

  This is caused by uploading `time.Now()` to the database.
  It saves only the local timestamp.
  It needs to be `time.Now().UTC()` for anything uploaded to the database.

- When I turn on my VPN and turn if off, the Twitch GQL requests work but the cloudfront requests don't work.
  I should try to understand why and fix it.
- Print debugging statements and errors separately.
- Add some private API so that I can configure the client ID and set of cloudfront domains at runtime.
  Maybe put these in a database so that I can retrieve them if the program restarts.
  Maybe have some additional service that monitors for client id and cloudfront domains to periodically update the database.
- Maybe update the list of domains with the domains retrieved via graphql and persist this to DB
