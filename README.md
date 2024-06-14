# misskey twins bot

## install

1. `vim config.toml`

``` toml
[misskey]
url = "misskey.until.tsukuba.one"
token = "{{ access token }}"
userId = "{{ userId }}" # maybe aid, aidx, objectid, or ulid format string probably
```

1. `go build main`
1. `./main -config config.toml`
