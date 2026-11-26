# Half-Life 1 map pipeline.  Every map lives in maps/<name>/ and builds into build/<name>/.
#   make build MAP=prairieloop     generate + compile + package  -> build/prairieloop/prairieloop.zip
#   make test  MAP=prairieloop     structure/limits, placements, wiring, walk routes
#   make views MAP=prairieloop     preview images  -> build/prairieloop/previews/
#   make video MAP=prairieloop     walkthrough mp4 + Discord-size copy (maps with video/walkthrough.py)
#   make all                       build + test every map
#   make readme                    regenerate README screenshot sections (map gallery, per-map grids)
#   make next AGENT=claude         the one thing this agent should do now (see docs/pipeline.md)
#   make ledger                    fetch the ledger branch -> build/ledger/ledger.sqlite + LEDGER.md
#   make quality MAP=x             satisfaction score (+ holdout pass rate when HOLDOUT_KEY is set)
# Long steps run under `timeout` and print progress; a silent process is a bug.

PY      ?= python3
MAP     ?=
MAPS    := $(filter-out _template,$(notdir $(wildcard maps/*)))
BD       = build/$(MAP)
TIMEOUT ?= 1800

.PHONY: build test views video all list check-map readme next ledger pipeline-test quality holdout-encrypt

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

next:
	@test -n "$(AGENT)" || (echo "usage: make next AGENT=claude|codex"; exit 2)
	$(PY) -m tools.pipeline.next $(AGENT)

ledger:
	$(PY) -m tools.pipeline.ledger fetch && $(PY) -m tools.pipeline.ledger build && $(PY) -m tools.pipeline.ledger render

pipeline-test:
	$(PY) -m tools.pipeline.test_gates

quality: check-map
	timeout 900 $(PY) -m tools.ci.quality measure $(MAP)

# Humans only: encrypt holdout/scenarios.json (never commit it) with the key stored as the HOLDOUT_KEY secret.
holdout-encrypt:
	@test -n "$$HOLDOUT_KEY" || (echo "set HOLDOUT_KEY"; exit 2)
	openssl enc -aes-256-cbc -pbkdf2 -salt -in holdout/scenarios.json -out holdout/scenarios.enc -pass env:HOLDOUT_KEY
	@echo "wrote holdout/scenarios.enc; delete holdout/scenarios.json"

readme:
	$(PY) -m tools.readme
