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

Note that `NULL` is not included in Postgres indices.
You must use something called a partial index to filter on `IS NULL` or `IS NOT NULL` efficiently.
This is not supported in Prisma.
So I'm just going with an additional boolean field to tell if an hls file fetch attempt has been made.

## Twitch

The Twitch GraphQL resolver for videos (in particular, past broadcasts) went down for a short period.
I should not trust the graphql API to work all the time.

I need to guarantee that the time I fetch a VOD is at least 30 minutes after the VOD ends.
This is because there seems to be a cron job (maybe a lambda service) that runs every 30 minutes to mute videos on the twitch servers.
This is described [here](https://www.reddit.com/r/osugame/comments/2cvspn/just_a_heads_up_twitchtv_is_now_muting_all_vods/).
The above is incorrect.
I guess videos are muted on the hour (e.g. 12:00AM, 01:00AM, ...).

This is annoying.
From `2022-12-01 08:57:50.429` to `2022-12-01 12:16:32.193`, a lot of the VODs just failed to fetch.
I should add some interal API to retry those.
7502 streams files were found.
2421 were not found.
This is 1500 more than average, so about 1500 streams failed out of around 10000.
Maybe replace the old vods queue with Apache Kafka or something.

Multiple Twitch VODs can have the same stream id.
This can happen, for example, if the streamer restarts the stream.
In this case, the time is `time.Second()` rather than `time.Unix()`.
In particular, see [this video](https://www.twitch.tv/videos/1671724933) from Twitch streamer `leagreasy`.
So for a primary key for a stream, I should use the composite key `(start_time, stream_id)` rather than just `stream_id`.

Cloudfront seems to default to a [rate limit](https://catalog.us-east-1.prod.workshops.aws/workshops/4d0b27bc-9f48-4356-8242-d13ca057fff2/en-US/application-layer-defense/rate-based-rules) of 2,000 requests per second. That is `6 2/3` requests per second.
I'm going to lower the number of HLS requests to 3 requests per second.
That comes out to 259200 streams per day.
Right now, the pace is 168000 streams per day that reach at least 10 views.
So this should be fine.

## Compression

I tried to migrate to Brotli compression.
But `mpv` seems to not support it, so I will not be using it.

## Live Vod Queue

Right now I have intermediate queue where everything is required to stay for 30 minutes, and I keep the live vod queue the at 5 minute intervals with VODs kept for 15 minutes.

This approach saves time for SQL fetched vods.
This will also solve my flooding problem after restarting.

An alternative approach is to include a second last interacted with field.
Then I only evict with this field is 45 minutes old.
But this fails to include the case where a streamer restarts in the stream.

## Debugging

All of the VODs older then `42 minutes + epsilon` should have `bytes_found` not be null.
I don't know why this is not the case.
Over `[now - 2 hrs, now - 1 hr]`, it's `3:3209`.
Over `[now - 10 hrs, now - 1 hr]`, it's `2239:45139`.

```SQL
SELECT COUNT(*) FROM streams WHERE bytes_found IS NULL AND max_views >= 10 AND last_updated_at BETWEEN NOW() - INTERVAL '10 hours' AND NOW() - INTERVAL '1 hour';
SELECT COUNT(*) FROM streams WHERE max_views >= 10 AND last_updated_at BETWEEN NOW() - INTERVAL '10 hours' AND NOW() - INTERVAL '1 hour';
```

Most of the VODs that are determined to be public should have `bytes_found = True`.
Right now, it's a ratio of `0:29046`.

```SQL
SELECT COUNT(*) FROM streams WHERE public = True AND bytes_found = False AND last_updated_at BETWEEN NOW() - INTERVAL '6 hours' AND NOW() - INTERVAL '43 minutes';
SELECT COUNT(*) FROM streams WHERE public = True AND last_updated_at BETWEEN NOW() - INTERVAL '6 hours' AND NOW() - INTERVAL '43 minutes';
```

## Current Status

- There are periods of 15 to 20 minutes where all the cloudfront URLs except for `https://vod-metro.twitch.tv/` work. In the past 12 hours, 1526 out of 49388 steams were public but with the bytes not found.
  So basically 3.1% of the streams have bytes not found as a result of these random periods.
- The current domains that work are

  ```text
  https://d1m7jfoe9zdc1j.cloudfront.net/
  https://d1mhjrowxxagfy.cloudfront.net/
  https://d1ymi26ma8va5x.cloudfront.net/
  https://d2nvs31859zcd8.cloudfront.net/
  https://d2vjef5jvl6bfs.cloudfront.net/
  https://d3vd9lfkzbru3h.cloudfront.net/
  https://dgeft87wbj63p.cloudfront.net/
  https://dqrpb9wgowsf5.cloudfront.net/
  https://ds0h3roq6wcgc.cloudfront.net/
  https://vod-metro.twitch.tv/
  https://vod-pop-secure.twitch.tv/
  https://vod-secure.twitch.tv/
  ```

- There are some stream id's that appear twice. None appear 3 times or more.
  A lot of them seem to be view botting.
  The one with the lowest views was at 49, but then it jumped up to 6885 after restarting.
  In the cases where the VODs are not deleted, the second stream ID the unix timestamp in the hls domain is replaced with the second of the start time.
  So I need to handle this case for streams that are restarted with the same stream ID.

  I find these streams with

  ```SQL
  WITH high_count AS (SELECT stream_id FROM streams GROUP BY stream_id HAVING COUNT(*) >= 2 ORDER BY COUNT(*) DESC LIMIT 50) SELECT streams.stream_id, streamer_login_at_start, max_views, public, sub_only, bytes_found, start_time, streamer_id, LEFT(title_at_start, 30), recording_fetched_at, last_updated_at, game_name_at_start FROM streams INNER JOIN high_count ON streams.stream_id = high_count.stream_id ORDER BY (streams.stream_id, last_updated_at);
  ```

  A corollary is if the stream is restarted a second time and it starts at the same second, I'm guessing the previous stream could be overwritten.
  Also, because these streams have the same stream id, the chat of the first stream is overwritten by the chat of the second stream.

- There are several cases where the unix time in the hls format is 1 minus the unix time of the TwitchGQL start time.
  This is strange. My current approach doesn't completely work 100% of the time because if the first request times out, it won't even try the second request.

## Unnecessary Cloudfront URLs

- https://d2e2de1etea730.cloudfront.net/. Nothing.
- https://d2aba1wr3818hz.cloudfront.net/. Nothing.
- https://d3c27h4odz752x.cloudfront.net/. Nothing.
- https://ddacn6pr5v0tl.cloudfront.net/. Nothing.
- https://d3aqoihi2n8ty8.cloudfront.net/. Returns an XML file with the following metadata.
  ```xml
  <Name>bits-assets</Name>
  <Prefix/>
  <Marker/>
  <MaxKeys>1000</MaxKeys>
  <IsTruncated>true</IsTruncated>
  ```

## DB Migrations

You see use `npx prisma migrate diff` as a tool to generate migrations, references [here](https://github.com/prisma/prisma/issues/8056#issuecomment-1034839831).

```bash
npx prisma migrate diff --from-empty --to-schema-datamodel ./schema.prisma --script
```

If you modify the Prisma schema, you can generate a new migration by comparing it to the previous migrations.
To requires a file `migration_lock.toml` to be in the migrations folder.
Right now the contents are

```toml
# Please do not edit this file manually
# It should be added in your version-control system (i.e. Git)
provider = "postgresql"
```

We can create to temporary databases and connect to them and generate down migrations from up migrations.

```bash
docker run --rm \
  --name shadow-db \
  -e POSTGRES_USER="user" \
  -e POSTGRES_PASSWORD="password" \
  -e POSTGRES_DB="db" \
  -p 8888:5432 \
  postgres
docker run --rm \
  --name shadow-db-2 \
  -e POSTGRES_USER="user" \
  -e POSTGRES_PASSWORD="password" \
  -e POSTGRES_DB="db" \
  -p 8889:5432 \
  postgres
psql postgresql://user:password@localhost:8888/db
psql postgresql://user:password@localhost:8889/db
npx prisma migrate diff --from-url postgresql://user:password@localhost:8889/db --to-url postgresql://user:password@localhost:8888/db --script
npx prisma migrate diff --from-migrations sqlc/migrations --to-schema-datamodel ./schema.prisma --shadow-database-url postgresql://user:password@localhost:8888/db --script
```

`pgcli` seems to be buggy. See [here](https://github.com/dbcli/pgcli/issues/1377).
In particular, it doesn't understand `--` when trying to create the `"streams"` table.

## Docker for Development

See [here](https://7thzero.com/blog/golang-w-sqlite3-docker-scratch-image) and [here](https://gist.github.com/zmb3/2bf29397633c9c9cc5125fdaa988c8a8)
for making statically linked Go binaries that include C dependencies.

To make building faster, use the [build cache](https://www.reddit.com/r/golang/comments/q7zppz/docker_cache_for_dependencies/) feature.

```bash
docker build -f ./docker/stringApi/Dockerfile -t twitch-vods-string-api:latest . --progress plain
docker build -f ./docker/scraper/Dockerfile -t twitch-vods-scraper:latest . --progress plain
docker build -f ./docker/caddy/Dockerfile -t twitch-vods-reverse-proxy:latest . --progress plain
```

Then we can create the containers.
To get the IP of a docker container from the host, see [here](https://stackoverflow.com/questions/17157721/how-to-get-a-docker-containers-ip-address-from-the-host).

```bash
# set PASSWORD env variable
source ./.env
DOCKER_POSTGRES_DB="postgresql://twitch-vods-admin:$PASSWORD@twitch-vods-db:5432/twitch-vods"
mkdir -p ~/docker/twitch-vods/twitch-vods-db
docker network create twitch-vods-network
docker run -d --restart always \
  --name twitch-vods-db \
  -e POSTGRES_USER="twitch-vods-admin" \
  -e POSTGRES_PASSWORD=$PASSWORD \
  -e POSTGRES_DB="twitch-vods" \
  -v ~/docker/twitch-vods/twitch-vods-db/data:/var/lib/postgresql/data \
  -v ~/docker/twitch-vods/twitch-vods-db/app:/home/app \
  --network twitch-vods-network \
  postgres:15
POSTGRES_DB="postgresql://twitch-vods-admin:$PASSWORD@$(docker inspect -f '{{range.NetworkSettings.Networks}}{{.IPAddress}}{{end}}' twitch-vods-db):5432/twitch-vods?sslmode=disable"
migrate -source file://sqlc/migrations -database $POSTGRES_DB up

# before running the stateless stuff below, migrate the data from the previous database
docker run -d --restart always \
  --name twitch-vods-string-api \
  -e DATABASE_URL=$DOCKER_POSTGRES_DB \
  -e CLIENT_URL="*" \
  --network twitch-vods-network \
  twitch-vods-string-api
docker run -d --restart always \
  --name twitch-vods-scraper \
  -e DATABASE_URL=$DOCKER_POSTGRES_DB \
  --network twitch-vods-network \
  twitch-vods-scraper
docker run -d --restart always \
  --name twitch-vods-reverse-proxy \
  --network twitch-vods-network \
  -v $PWD/caddy/dev.Caddyfile:/etc/caddy/Caddyfile:ro \
  -p 3000:3000 \
  twitch-vods-reverse-proxy
```

## Migrating from Old Docker to New Docker

I have a docker container called `sensitive_data` on my default bridge network with the port mapping `-p 5432:5432`.
I want to migrate it to my `twitch-vods-network` bridge network.

First, we configure a docker container on the default bridge network and bind its contents to the host.
I need to user a debugger container because I need to use `pg_dump` for Postgres 15, and the version installed on my system is for Postgres 14.

[This answer](https://stackoverflow.com/a/59307721) basically shows how to create a backup with `pg_dump` that is compatible with `pg_restore`.
Basically, you have to use `--format=custom` which is the same as `-F c`.
Then you can restore it with `pg_restore`.
By default, it uses GZIP compression.
In Postgres 16, it will get ZSTD compression.
There are some notes [here](https://stackoverflow.com/questions/15692508/a-faster-way-to-copy-a-postgresql-database-or-the-best-way) on how to make this faster.
For absolute speed, it's probably better to set `-Z0` and then `rsync` with `--compress-choice=zstd --compress-level=3 --checksum-choice=xxh3`.
See [here](https://news.ycombinator.com/item?id=26371810) for those `rsync` options.
This takes more storage space.

```bash
# From the host
mkdir -p ~/docker/twitch-vods/twitch-vods-db-debugger
docker run --rm \
  -e POSTGRES_PASSWORD=password \
  --name twitch-vods-db-debugger \
  -v ~/docker/twitch-vods/twitch-vods-db-debugger/data:/var/lib/postgresql/data \
  -v ~/docker/twitch-vods/twitch-vods-db-debugger/app:/home/app \
  postgres:15
docker exec -it twitch-vods-db-debugger /bin/bash

# Now in the twitch-vods-db-debugger
apt-get update && apt-get -y upgrade && apt-get -y install curl iproute2 net-tools
route # 172.17.0.1
pg_dump --format=custom --file /home/app/backup.dump postgresql://govods:password@172.17.0.1:5432/twitch

# Back in the host
sudo mv ~/docker/twitch-vods/twitch-vods-db-debugger/app/backup.dump ~/docker/twitch-vods/twitch-vods-db/app
docker stop twitch-vods-db-debugger
sudo rm -rf ~/docker/twitch-vods/twitch-vods-db-debugger/
docker exec -it twitch-vods-db /bin/bash

## Now in the twitch-vods-db
# set PASSWORD
DOCKER_POSTGRES_DB="postgresql://twitch-vods-admin:$PASSWORD@localhost:5432/twitch-vods"
pg_restore --verbose --clean --no-owner --dbname $DOCKER_POSTGRES_DB /home/app/backup.dump
```

## Copying to Remote

For debugging:

```bash
# in the host
mkdir -p ~/docker/twitch-vods/twitch-vods-debugger
source .env
docker run --rm -it \
  --name twitch-vods-debugger \
  -v ~/docker/twitch-vods/twitch-vods-debugger/app:/home/app \
  -e POSTGRES_PASSWORD=password \
  -e DATABASE_URL=$DOCKER_POSTGRES_DB \
  --network twitch-vods-network \
  postgres:15
docker exec -it twitch-vods-debugger /bin/bash

# in the container
pg_dump -Z0 -Fc -f /home/app/backup.dump $DATABASE_URL

# in the host
rsync -avzhP \
  --compress-choice=zstd \
  --compress-level=1 \
  --checksum-choice=xxh3 \
  --rsync-path $RSYNC_PATH \
  ~/docker/twitch-vods/twitch-vods-debugger/app/backup.dump $REMOTE_USER:docker/backup.dump
```

## Benchmarking

I used to use `ab` for load testing. I tried out `wrk`, but it seems to suffer from coordinated omission.
See [here](https://news.ycombinator.com/item?id=10486215) for a definition.
Instead, use something like `wrk2` or `vegeta` which make requests at a fixed rate.
Further discussion is [here](https://lobste.rs/s/mqxwuc/what_s_your_preferred_tool_for_load).

```bash
echo "GET http://localhost:3000/all/private/sub" | vegeta attack -duration 1000ms -rate 40000 | vegeta report --type=text
```

If the duration is set too high, `vegeta` will open too many ports and get an error message.
Recall that a connection mathematically is a tuple `(server_ip, server_port, client_ip, client_port)`.
In Linux, there are `64K` ports, but practically it's more like `40K`.
So when running on a single machine, the value `duration * rate` should be at most `40000`.
See an approximation of the number of connections with `netstat -atn | wc -l`.

```text
Get "http://localhost:3000/all/private/sub": dial tcp 0.0.0.0:0->[::1]:3000: bind: address already in use
```

View the CPU and memory usage of each docker container with `docker stats`.
The memory usage of `twitch-vods-string-api` after being load tested is really high.
Right now it's around 782 MiB, which is more than the Postgres database at 680 MiB.
I think this is because of `pgx`. See [this issue](https://github.com/jackc/pgx/issues/1127) and [this issue](https://github.com/jackc/pgx/issues/845).
It seems to have been [resolved](https://github.com/jackc/pgx/blob/master/CHANGELOG.md#reduced-memory-usage-by-reusing-read-buffers) in `pgx v5`.

## Backpressure

- https://www.youtube.com/watch?v=m64SWl9bfvk&t=1676s
- https://github.com/platinummonkey/go-concurrency-limits

Basically, I'm using a pool of goroutines.
When it becomes exhausted, it returns error responses.
I use a bunch of `select`s.
There's probably a simpler implementation, but this was the first thing that came to my mind.
Before, the latency could theoretically become unbounded.
If I threw 50000 requests per second, all the ports would become used up and the average latency would exceed a second.
Now with 50000 requests per second, there are 43000 responses per second, with 7500 responses per second being successful and 35000 responses per second failing.
Now, the latency is reasonable.

```bash
echo "GET http://localhost:3000/all/private/sub" | vegeta attack -duration 5000ms -rate 50000 | vegeta report --type=text
```

## Network Debugging

To see the connections in the scraper container, use this.

```bash
sudo nsenter -t $(docker inspect -f '{{.State.Pid}}' twitch-vods-scraper) -n netstat
```

This comes from [here](https://stackoverflow.com/questions/40350456/docker-any-way-to-list-open-sockets-inside-a-running-docker-container).
First, when it only uses the Twitch GQL endpoint, it shows the Postgresql address and an IP address associated with Fastly.
See `whois $IP_ADDRESS` and `dig -x $IP_ADDRESS`.

## Robust HTTP

For random 16 minute intervals, the hls fetcher http client would return `context timeout` errors.
Basically, it has to do with how Go [implements](https://github.com/golang/go/issues/36026#issuecomment-569029370) HTTP/2 and how `ip_retries2` [works](https://blog.cloudflare.com/when-tcp-sockets-refuse-to-die/).
To replicate it, just [block](https://github.com/golang/go/issues/30702) the underlying TCP connection with `iptables`.
The solution is to be mindful of what Go [timeouts](https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/) actually do.
I should try to make [adverserial](https://blog.cloudflare.com/exposing-go-on-the-internet/) HTTP clients that don't close themselves in order to test the string API.

Note that when turning on a VPN, the HTTP clients in the docker containers will not be able to establish any [connections](https://serverfault.com/questions/895278/not-able-to-access-to-the-internet-in-a-container-on-a-vpn).

## Domain and Cloudflare Setup

Buy domain from porkbun.
Alternatively, see [here](https://domcomp.com/) for domain deals.
Create cloudflare website portal.
In porkbun dashbaord, set cloudflare nam servers as authoritative name servers.
Delete all DNS settings.
Then set everything up with terraform.

## Caddy

To setup Docker, Caddy, and Cloudflare, see [here](https://caddy.community/t/setting-up-cloudflare-with-caddy/13911).

## TODO

- Use caddy as a reverse proxy. Don't use it to serve the static SPA. Just use some SAAS that does this.
- Deploy with Terraform and cloudflare.
  Use [Authenticated origin pulls](https://caddy.community/t/setting-up-cloudflare-with-caddy/13911/6) and the cloudflare [module](https://github.com/caddy-dns/cloudflare) for caddy.
- `pgx v4` uses too much memory. Migrate to `pgx v5`.
  `sqlc v16` is not compatible with `pgx v5`.
  Support has been merged into the main branch. I should build `sqlc` from source and set it to generate `pgx v5` code.
- I'm using `go-libdeflate` which is using version 1.6 of `libdeflate`.
  The latest version of `libdeflate` is 1.15
  It caused [this issue](https://github.com/4kills/go-libdeflate/issues/13).
  Either make a pull request or fork the project to update the version of `libdeflate` in `go-libdeflate`.
- I'm maintaining an infinite for loop.
  I should check if all the goroutines are closed using some tool to inspect the program internals.
- Maybe add some private API so that I can configure the client ID and set of cloudfront domains at runtime.
  Maybe put these in a database so that I can retrieve them if the program restarts.
  Maybe have some additional service that monitors for client id and cloudfront domains to periodically update the database.
- Maybe update the list of domains with the domains retrieved via graphql and persist this to DB
