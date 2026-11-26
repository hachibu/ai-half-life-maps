<!--
Write this for a person who hasn't seen your code or the chat that produced it.
Plain language, short sentences, no internal jargon. Replace every comment; delete sections only where noted.
-->

## Summary

<!-- 1-3 sentences: what this pull request does and why.
Example: "Adds a frozen pond to the park zone of prairieloop. Shooting the ice drops players into a weapons cache." -->

## What a player will notice

<!-- Bullets, from the player's point of view. Say where to go to see it (a spawn or landmark), what they see,
and what moves or reacts. For tools-only changes, write "Nothing in game" and explain the effect on agents instead. -->

- 

## Screenshots

<!-- CI posts before/after screenshots as a comment on this pull request a few minutes after each push
(before = main, after = this branch; changed views are shown first).

Every change a player can see needs at least one shot aimed at it. Add a view to your zone's VIEWS
(or maps/<map>/tests/views.py) named for what it shows, e.g. ("pond_from_bridge", eye, target).
Run `make views MAP=<map>` and look at each image yourself before pushing: nothing black, no missing
textures, camera not inside a wall.

List the shots a reviewer should look at and what each one shows. For moving or interactive things
(doors, lifts, breakables), say what state the still shows and what happens next. -->

- `view_name`: 

## Changes

<!-- One row per file or zone: what changed and why. Call out anything surprising: a limit you're close to,
a workaround, a shared file you had to touch, something you removed. -->

| File or zone | What changed and why |
|---|---|
|  |  |

## Check results

<!-- Paste the end of `make test MAP=<map>`: the limits table and the route lines. -->

```
paste here
```

- [ ] `make test` passes locally and CI is green
- [ ] every new walkable area has a route in `ROUTES`
- [ ] every visible change has a named view, listed under Screenshots, and I looked at the images
- [ ] only original art, names and sounds

## Lessons

<!-- Optional: lessons from AGENTS.md you applied, and any you added or updated in docs/lessons/. -->

## Not verified / risks

<!-- What you couldn't check, e.g. "not played in the real game", "lift timing only seen in stills".
Write "Nothing" only if that's true. -->

