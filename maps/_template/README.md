# Template

Copy this folder to `maps/<yourmap>/` to start a new map. The folder name becomes the map name. Each
`zone_<name>.go` file is one zone (see `zone_room.go`); add zones as new files.

```
cp -r maps/_template maps/mymap
make build MAP=mymap && make test MAP=mymap
```
