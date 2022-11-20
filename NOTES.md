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

In order to update a bunch of rows using a subquery, see [here](https://stackoverflow.com/a/45465626).
This seems like the most modern approach.

## Pagination in Postgres

I want to implement some kind of cursor based pagination.
[This article](https://www.citusdata.com/blog/2016/03/30/five-ways-to-paginate/) describes some approaches.
The limit-offset approach is the most naive, and the slowest.
So if we want to have a list of pages like in old.reddit, it might be better to maintain an in-memory index and then this index to fetch with keyset pagination.

Cursors seem complicated for me.

I like the keyset pagination approach.
The solution described in the article is too simple.

For example, it looks like

```sql
CREATE INDEX "streams_bytes_found_recording_fetched_at_id_idx" ON "streams"("bytes_found", "recording_fetched_at", "id");

SELECT
  id, stream_id, recording_fetched_at
FROM
  streams
WHERE
  bytes_found = True
  AND (recording_fetched_at, id) < ($1, $2)
ORDER BY
  bytes_found DESC, recording_fetched_at DESC, id DESC
LIMIT 20;
```

I'm not sure if I can replace the order-by condition `bytes_found DESC, recording_fetched_at DESC, id DESC` with `recording_fetched_at DESC, id DESC` since the first value is fixed in the query.

Here are some links.

- https://use-the-index-luke.com/no-offset. This article gives a basic introduction to pagination in SQL.
- https://use-the-index-luke.com/sql/partial-results/fetch-next-page. This gives a more in-depth explanation. In particular, the equivalent logical condition is not executed the same.
- https://vladmihalcea.com/sql-seek-keyset-pagination/. I like how this article describes how to using the planning tool to assess the performance of a query.
  I dislike how they say "row value expression is equivalent" to the expanded expression.
  It is not executed the same. They are logically equivalent, but not functionally equivalent.
- https://www.postgresql.org/message-id/20191117182408.GA13566@fetter.org.
  This email exchange describes how they try to add support for reverse collation, to make keyset pagination cover more cases. It would be nice to do something like `(recording_fetched_at, DESC id) < ($1, $2)` so that I can use fewer indices, but that feature does not seem to exist.
- https://stackoverflow.com/questions/58942808/how-to-keypaginate-uuid-in-postgresql and https://stackoverflow.com/questions/70519518/is-there-any-better-option-to-apply-pagination-without-applying-offset-in-sql-se.
  These are just some stackoverflow links.

I just tried it.

- `bytes_found DESC, recording_fetched_at DESC, id DESC` is 0.01s.
- `recording_fetched_at DESC, id DESC` is 0.01s.
- `bytes_found DESC, recording_fetched_at DESC` is 0.01s.
- `bytes_found, recording_fetched_at DESC, id DESC` is 0.2s.
- `bytes_found, recording_fetched_at DESC` is 0.2s.

Basically, I don't need to include `bytes_found`, but if I do include it, then it must follow the order of the index.

We can use `EXPLAIN` to understand what a query is doing.
See these links for explanation:

- https://www.postgresql.org/docs/current/using-explain.html
- https://scalegrid.io/blog/postgres-explain-cost/

Running

```sql
EXPLAIN SELECT id, stream_id, recording_fetched_at, max_views, streamer_login_at_start FROM streams WHERE bytes_found = True AND (recording_fetched_at, id) < ('2022-11-15 08:17:47.118', 'fe5b6c61-bc22-41a5-9674-7d85055519fc') ORDER BY bytes_found DESC, recording_fetched_at DESC LIMIT 50;
```

```text
Limit  (cost=0.42..79.89 rows=50 width=56)
  ->  Index Scan Backward using streams_bytes_found_recording_fetched_at_id_idx on streams  (cost=0.42..104199.46 rows=65564 width=56)
        Index Cond: ((bytes_found = true) AND (ROW(recording_fetched_at, id) < ROW('2022-11-15 08:17:47.118'::timestamp without time zone, 'fe5b6c61-bc22-41a5-9674-7d85055519fc'::uuid)))
```

So the estimated cost is 79.89.

Running

```sql
EXPLAIN SELECT id, stream_id, recording_fetched_at, max_views, streamer_login_at_start FROM streams WHERE bytes_found = True AND (recording_fetched_at, id) < ('2022-11-15 08:17:47.118', 'fe5b6c61-bc22-41a5-9674-7d85055519fc') ORDER BY bytes_found, recording_fetched_at DESC LIMIT 50;
```

```text
Limit  (cost=53488.54..53530.05 rows=50 width=56)
  ->  Incremental Sort  (cost=53488.54..107877.63 rows=65522 width=56)
        Sort Key: bytes_found, recording_fetched_at DESC
        Presorted Key: bytes_found
        ->  Index Scan using streams_bytes_found_recording_fetched_at_id_idx on streams  (cost=0.42..103711.77 rows=65522 width=56)
              Index Cond: ((bytes_found = true) AND (ROW(recording_fetched_at, id) < ROW('2022-11-15 08:17:47.118'::timestamp without time zone, 'fe5b6c61-bc22-41a5-9674-7d85055519fc'::uuid)))
```

## Twitch

The Twitch GraphQL resolver for videos (in particular, past broadcasts) went down for a short period.
I should not trust the graphql API to work all the time.

## Compression

I tried to migrate to Brotli compression.
But `mpv` seems to not support it, so I will not be using it.

## TODO

- Some of the segments in some videos are not loading.
  It seems like they are muted, but the m3u8 file did not include that information.
  Maybe I should wait longer before fetching the video.
  This [link](https://www.reddit.com/r/osugame/comments/2cvspn/just_a_heads_up_twitchtv_is_now_muting_all_vods/) seems to explain it.
  I should add a queue in between the old vods queue and the live vods queue.
  That queue should keep each video in for at least 30 minutes.
  Additionally, I should remove the wait time for the live vods queue from 15 minutes to 10 minutes.
  The initial vods queue should contain all the vods from the last `buffer_ratio * (old_vods_eviction_time + intermediate_queue_time)` minutes.
  Also, I should allow an entry in the intermediate queue to be removed if an updated version is found in the live vods queue.
  This will also solve my flooding problem mentioned below.
- When I restart the scraper after 15 minutes, the old vods queue is flooded with all the the vods.
  I should add a new separate field called `ScraperLastFetchedTime` that is set when I fetch from the database or from the Twitch GQL API.
  This field should be used to evict from the database.
- I'm maintaining an infinite for loop.
  I should check if all the goroutines are closed using some tool to inspect the program internals.
- Return VOD to VODs list if it is still live using the GraphQL client to check.
  Alternatively, when a live vod is fetched, check if it's in the old vod queue. If it's there, remove it from the old vod queue.
- The m3u8 cloudfront links stopped working for about 5 minutes.
  As a result, like 60 public videos didn't have their m3u8 contents fetched.
  I'm not sure how to handle this.
- When I turn on my VPN and turn if off, the Twitch GQL requests work but the cloudfront requests don't work.
  I should try to understand why and fix it.
- Print debugging statements and errors separately.
- Add some private API so that I can configure the client ID and set of cloudfront domains at runtime.
  Maybe put these in a database so that I can retrieve them if the program restarts.
  Maybe have some additional service that monitors for client id and cloudfront domains to periodically update the database.
- Maybe update the list of domains with the domains retrieved via graphql and persist this to DB
