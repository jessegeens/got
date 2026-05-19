# Got
![coverage](https://raw.githubusercontent.com/jessegeens/got/badges/.badges/main/coverage.svg)

Got is a tiny Git client, written in Go. It is a loose translation of [wyag](https://wyag.thb.lt); but in Go instead of Python.

The following commands are supported:
- `got add <path>...`
- `got checkout <reference>`
- `got commit [-m message]`
- `got init [path]`
- `got log [commit]`
- `got rm <path>`
- `got status`
- `got tag [-annotate] <name> <object>`


As well as the following "plumbing" commands:
- `got cat-file <hash>`
- `got check-ignore <path>...`
- `got hash-object [-w] <path> <type>`
- `got ls-files [-verbose]`
- `got ls-tree [-r] <tree>`
- `got rev-parse <type> <name>`
- `got show-ref`

Note that got does not support pack files!