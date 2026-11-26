# Half-Life 1 map pipeline.  Every map lives in maps/<name>/ and builds into build/<name>/.
#   make build MAP=prairieloop     generate + compile + package  -> build/prairieloop/prairieloop.zip
#   make test  MAP=prairieloop     structure/limits, placements, wiring, walk routes
#   make views MAP=prairieloop     preview images  -> build/prairieloop/previews/
#   make video MAP=prairieloop     walkthrough mp4 + Discord-size copy (maps with video/walkthrough.py)
#   make all                       build + test every map
# Long steps run under `timeout` and print progress; a silent process is a bug.

PY      ?= python3
MAP     ?=
MAPS    := $(filter-out _template,$(notdir $(wildcard maps/*)))
BD       = build/$(MAP)
TIMEOUT ?= 1800

.PHONY: build test views video all list check-map

check-map:
	@test -n "$(MAP)" || (echo "usage: make <target> MAP=<one of: $(MAPS)>"; exit 2)
	@test -d maps/$(MAP) || (echo "no such map: maps/$(MAP)"; exit 2)

list:
	@echo $(MAPS)

build: check-map
	timeout $(TIMEOUT) $(PY) maps/$(MAP)/build.py
	timeout $(TIMEOUT) $(PY) -u -m tools.bsp.compile $(BD)/$(MAP).map $(BD)/$(MAP).wad $(BD)/$(MAP).bsp
	$(PY) -m tools.package $(MAP)

test: check-map
	$(PY) -m tools.checks.structure $(BD)/$(MAP).bsp
	$(PY) -m tools.checks.placements $(BD)/$(MAP).bsp
	$(PY) -m tools.checks.wiring $(BD)/$(MAP).bsp
	@if [ -f maps/$(MAP)/tests/routes.py ]; then timeout 900 $(PY) maps/$(MAP)/tests/routes.py $(BD)/$(MAP).bsp; fi

views: check-map
	timeout 900 $(PY) -m tools.render.views $(MAP)

video: check-map
	@test -f maps/$(MAP)/video/walkthrough.py || (echo "$(MAP) has no video/walkthrough.py"; exit 2)
	timeout 900  $(PY) maps/$(MAP)/video/walkthrough.py check
	timeout 3600 $(PY) maps/$(MAP)/video/walkthrough.py render
	timeout 900  $(PY) maps/$(MAP)/video/walkthrough.py audio
	cd $(BD)/video && ls chunk_??.mp4 | sort | sed "s/^/file '/; s/$$/'/" > list.txt \
	  && ffmpeg -y -loglevel error -f concat -safe 0 -i list.txt -c copy video_only.mp4 \
	  && ffmpeg -y -loglevel error -i video_only.mp4 -i audio.wav -map 0:v -map 1:a -c:v copy -c:a aac -b:a 160k -shortest ../$(MAP)_walkthrough.mp4 \
	  && ffmpeg -y -loglevel error -i ../$(MAP)_walkthrough.mp4 -c:v libx264 -preset medium -b:v 1450k -pass 1 -an -f mp4 /dev/null \
	  && ffmpeg -y -loglevel error -i ../$(MAP)_walkthrough.mp4 -c:v libx264 -preset medium -b:v 1450k -pass 2 -pix_fmt yuv420p -c:a aac -b:a 96k -movflags +faststart ../$(MAP)_walkthrough_discord.mp4
	@ls -la $(BD)/$(MAP)_walkthrough*.mp4

all:
	@set -e; for m in $(MAPS); do $(MAKE) build MAP=$$m && $(MAKE) test MAP=$$m; done
