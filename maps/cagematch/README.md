# Cage Match

An octagonal pit, 176 units deep, behind a see-through chain-link fence with padded posts. A balcony ring
overhangs it; walk off the striped edge to drop in. Nobody climbs back out: walls are taller than any jump,
and a downward push in the drop hole stops gauss jumps and grenade boosts.

- **4 pit spawns** get every weapon in the game (a `game_player_equip` fired by the spawn point).
- **12 balcony spawns** get nothing, not even the crowbar.
- 12 weapons and 9 ammo/health/battery pickups lie on the pit floor.

Console: `map cagematch`

## Build it

```
make build MAP=cagematch && make test MAP=cagematch
```
The playable zip lands in `build/cagematch/cagematch.zip`.
