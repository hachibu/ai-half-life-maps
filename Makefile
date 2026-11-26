# Half-Life 1 map pipeline.  Every map lives in maps/<name>/ and builds into build/<name>/.
#   make build MAP=prairieloop     generate + compile (Go: tools/bsp/bspc) + package -> build/prairieloop/prairieloop.zip
#   make golden                    compiler + scorer output (template map, prairieloop) must match the committed golden values
#   make golden-update             re-record the golden values after an intended output change (commit them)
#   make lint                      golangci-lint: gofmt, go vet, staticcheck, errcheck, unused (tools/.golangci.yml)
#   make test  MAP=prairieloop     structure/limits, placements, wiring, walk routes
#   make views MAP=prairieloop     preview images  -> build/prairieloop/previews/
#   make video MAP=prairieloop     walkthrough mp4 + Discord-size copy (maps with video/walkthrough.py)
#   make all                       build + test every map
#   make readme                    regenerate README screenshot sections (map gallery, per-map grids)
#   make ledger                    fetch the ledger branch -> build/ledger/ledger.sqlite + LEDGER.md
#   make quality MAP=x             satisfaction score (+ holdout pass rate when HOLDOUT_KEY is set)
# Long steps run under `timeout` and print progress; a silent process is a bug.

PY      ?= python3
MAP     ?=
MAPS    := $(filter-out _template,$(notdir $(wildcard maps/*)))
BD       = build/$(MAP)
TIMEOUT ?= 1800

.PHONY: bspc lint golden golden-out golden-update build test views video all list check-map readme ledger pipeline-test quality holdout-encrypt

check-map:
	@test -n "$(MAP)" || (echo "usage: make <target> MAP=<one of: $(MAPS)>"; exit 2)
	@test -d maps/$(MAP) || (echo "no such map: maps/$(MAP)"; exit 2)

list:
	@echo $(MAPS)

BSPC     = build/bin/bspc

SATISFY  = build/bin/satisfy

$(BSPC): $(wildcard tools/bsp/bspc/*.go) go.mod
	go build -o $(BSPC) ./tools/bsp/bspc

$(SATISFY): $(wildcard tools/checks/satisfy/*.go tools/bsp/bspfile/*.go tools/checks/walker/*.go) go.mod
	go build -o $(SATISFY) ./tools/checks/satisfy

MAPCHECK = build/bin/mapcheck

$(MAPCHECK): $(wildcard tools/checks/mapcheck/*.go tools/bsp/bspfile/*.go tools/checks/walker/*.go) go.mod
	go build -o $(MAPCHECK) ./tools/checks/mapcheck

bspc: $(BSPC) $(SATISFY) $(MAPCHECK)

build: check-map $(BSPC)
	timeout $(TIMEOUT) $(PY) maps/$(MAP)/build.py
	timeout $(TIMEOUT) $(BSPC) $(BD)/$(MAP).map $(BD)/$(MAP).wad $(BD)/$(MAP).bsp
	$(PY) -m tools.package $(MAP)

test: check-map $(MAPCHECK)
	$(MAPCHECK) structure $(BD)/$(MAP).bsp
	$(MAPCHECK) placements $(BD)/$(MAP).bsp
	$(MAPCHECK) wiring $(BD)/$(MAP).bsp
	@if [ -f maps/$(MAP)/tests/routes.py ]; then \
	  ROUTES_JSON=1 $(PY) maps/$(MAP)/tests/routes.py > $(BD)/routes.json && \
	  timeout 900 $(MAPCHECK) routes $(BD)/$(MAP).bsp $(BD)/routes.json; fi

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

ledger:
	$(PY) -m tools.pipeline.ledger fetch && $(PY) -m tools.pipeline.ledger build && $(PY) -m tools.pipeline.ledger render

pipeline-test:
	$(PY) -m tools.pipeline.test_gates

quality: check-map $(SATISFY)
	timeout 900 $(PY) -m tools.ci.quality measure $(MAP)

# Humans only: encrypt holdout/scenarios.json (never commit it) with the key stored as the HOLDOUT_KEY secret.
holdout-encrypt:
	@test -n "$$HOLDOUT_KEY" || (echo "set HOLDOUT_KEY"; exit 2)
	openssl enc -aes-256-cbc -pbkdf2 -salt -in holdout/scenarios.json -out holdout/scenarios.enc -pass env:HOLDOUT_KEY
	@echo "wrote holdout/scenarios.enc; delete holdout/scenarios.json"

readme:
	$(PY) -m tools.readme

# Go is the only compiler and scorer, so their output is pinned to committed golden values: the compiled BSP's
# SHA-256 and the satisfaction result (seed 4242) for the template map and prairieloop (a real map whose metrics
# aren't all 1.0). Any change to that output has to be deliberate (`make golden-update`) and shows in the PR diff.
GOLDEN_MAPS = _template prairieloop
GOLDEN_BSP  = tools/bsp/bspc/golden.sha256
GOLDEN_SAT  = tools/checks/satisfy/golden.json
GO_OUT      = build/golden

golden-out: $(BSPC) $(SATISFY)
	@mkdir -p $(GO_OUT); rm -f $(GO_OUT)/*
	@set -e; for m in $(GOLDEN_MAPS); do \
	  echo "golden: $$m"; \
	  timeout 900 $(PY) maps/$$m/build.py > /dev/null; \
	  timeout 900 $(BSPC) build/$$m/$$m.map build/$$m/$$m.wad $(GO_OUT)/$$m.bsp > /dev/null; \
	  (cd $(GO_OUT) && sha256sum $$m.bsp) >> $(GO_OUT)/golden.sha256; \
	  SATISFACTION_SEED=4242 timeout 900 $(SATISFY) $(GO_OUT)/$$m.bsp --json $(GO_OUT)/$$m.sat.json > /dev/null; \
	done
	$(PY) -c "import json, sys; out = {}; \
	  [out.__setitem__(m, {k: v for k, v in json.load(open('$(GO_OUT)/%s.sat.json' % m)).items() if k != 'seconds'}) for m in sys.argv[1:]]; \
	  print(json.dumps(out, indent=1, sort_keys=True))" $(GOLDEN_MAPS) > $(GO_OUT)/golden.json

golden: golden-out
	@diff -u $(GOLDEN_BSP) $(GO_OUT)/golden.sha256 && echo "compiler golden: identical" || \
	  (echo "compiler output changed. If that is intended, run make golden-update and commit the result."; exit 1)
	@diff -u $(GOLDEN_SAT) $(GO_OUT)/golden.json && echo "scorer golden: identical" || \
	  (echo "scorer output changed. If that is intended, run make golden-update and commit the result."; exit 1)

golden-update: golden-out
	cp $(GO_OUT)/golden.sha256 $(GOLDEN_BSP)
	cp $(GO_OUT)/golden.json $(GOLDEN_SAT)
	@echo "updated $(GOLDEN_BSP) and $(GOLDEN_SAT); commit them and say in the PR why the output changed"

# golangci-lint (config: tools/.golangci.yml) covers gofmt, go vet, staticcheck, errcheck, unused and ineffassign.
# Install v2.15.0, the version CI pins: https://golangci-lint.run/docs/welcome/install/local/
lint:
	@if command -v golangci-lint >/dev/null; then timeout 600 golangci-lint run -c tools/.golangci.yml ./...; else \
	  echo "golangci-lint not installed (CI runs it); falling back to gofmt + go vet"; \
	  test -z "$$(gofmt -l tools)" || (echo "gofmt needed:"; gofmt -l tools; exit 1); \
	  go vet ./...; fi
