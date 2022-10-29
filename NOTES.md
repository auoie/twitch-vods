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
