# Half-Life 1 map pipeline.  Every map lives in maps/<name>/ and builds into build/<name>/.
#   make build MAP=prairieloop     generate + compile (SDHLT: full VIS + full lighting) + package -> build/prairieloop/prairieloop.zip
#   make build MAP=x LIGHT=quick VIS=fast   a much faster draft compile while iterating locally
#   make golden                    the template map compiled by SDHLT must match its committed SHA-256
#   make golden-update             re-record the golden values after an intended output change (commit them)
#   make lint                      golangci-lint: gofmt, go vet, staticcheck, errcheck, unused (tools/.golangci.yml)
#   make test  MAP=prairieloop     structure/limits, placements, wiring, walk routes
#   make views MAP=prairieloop     preview images  -> build/prairieloop/previews/
#   make video MAP=prairieloop     walkthrough mp4 + Discord-size copy (maps with video/walkthrough.py)
#   make all                       build + test every map
#   make readme [V=<commit>]       regenerate README screenshot sections; V (a previews commit) makes browsers refetch images
# Long steps run under `timeout` and print progress; a silent process is a bug.

PY      ?= python3
MAP     ?=
MAPS    := $(filter-out _template,$(notdir $(wildcard maps/*)))
BD       = build/$(MAP)
TIMEOUT ?= 1800

.PHONY: bspc sdhlt lint golden golden-out golden-update build test views video all check-map readme

check-map:
	@test -n "$(MAP)" || (echo "usage: make <target> MAP=<one of: $(MAPS)>"; exit 2)
	@test -d maps/$(MAP) || (echo "no such map: maps/$(MAP)"; exit 2)

# SDHLT (https://github.com/seedee/SDHLT): the GoldSrc compiler suite, built from the commit in tools/bsp/sdhlt.rev.
# It is GPLv2 + Valve's SDK EULA, so it is fetched and built here, never committed. Needs git, cmake and a C++ compiler.
SDHLT_REV := $(shell cat tools/bsp/sdhlt.rev)
SDHLT     = $(CURDIR)/build/sdhlt/$(SDHLT_REV)/tools
# PRs and releases build what players get: full VIS and full lighting. LIGHT=quick VIS=fast is a draft mode for
# local iteration. Full VIS stays fast because the maps put small brushes in func_detail (tools/mapkit.py).
LIGHT    ?= full
VIS      ?= full
BSPFLAGS  = -maxnodesize 2048
ifeq ($(VIS),full)
VISFLAGS  =
else
VISFLAGS  = -fast
endif
ifeq ($(LIGHT),full)
RADFLAGS  =
else
RADFLAGS  = -fast -bounce 2
endif

$(SDHLT)/sdHLRAD: tools/bsp/sdhlt.rev
	rm -rf build/sdhlt/$(SDHLT_REV) && mkdir -p build/sdhlt
	git clone -q https://github.com/seedee/SDHLT build/sdhlt/$(SDHLT_REV)
	cd build/sdhlt/$(SDHLT_REV) && git -c advice.detachedHead=false checkout -q $(SDHLT_REV) \
	  && cmake -S . -B b -DCMAKE_BUILD_TYPE=Release > /dev/null && cmake --build b -j > /dev/null
	@test -x $(SDHLT)/sdHLRAD && touch $(SDHLT)/sdHLRAD

sdhlt: $(SDHLT)/sdHLRAD

# compile MAP_DIR NAME: the four SDHLT steps on NAME.map (+ NAME.wad) in MAP_DIR, writing NAME.bsp there.
# -nowadtextures embeds every texture in the .bsp, so the map needs no separate .wad.
define compile
	cd $(1) && timeout $(TIMEOUT) $(SDHLT)/sdHLCSG -nowadtextures $(2) > $(2).csg.log \
	  && timeout $(TIMEOUT) $(SDHLT)/sdHLBSP $(BSPFLAGS) $(2) > $(2).bsp.log \
	  && timeout $(TIMEOUT) $(SDHLT)/sdHLVIS $(VISFLAGS) $(2) > $(2).vis.log \
	  && timeout $(TIMEOUT) $(SDHLT)/sdHLRAD $(RADFLAGS) $(2) > $(2).rad.log \
	  || (tail -40 $(2).log; exit 1)
	@cd $(1) && for t in csg bsp vis rad; do printf '  %-4s %s\n' $$t "$$(grep -h 'seconds elapsed' $(2).$$t.log | tail -1)"; done
endef

MAPCHECK = build/bin/mapcheck

$(MAPCHECK): $(wildcard tools/checks/mapcheck/*.go tools/bsp/bspfile/*.go tools/checks/walker/*.go) go.mod
	go build -o $(MAPCHECK) ./tools/checks/mapcheck

RENDER   = build/bin/render
BSPINFO  = build/bin/bspinfo
CI       = build/bin/ci

$(RENDER): $(wildcard tools/render/render/*.go tools/render/renderer/*.go tools/bsp/bspfile/*.go) go.mod
	go build -o $(RENDER) ./tools/render/render

$(BSPINFO): $(wildcard tools/bsp/bspinfo/*.go tools/bsp/bspfile/*.go) go.mod
	go build -o $(BSPINFO) ./tools/bsp/bspinfo

$(CI): $(wildcard tools/ci/ci/*.go) go.mod
	go build -o $(CI) ./tools/ci/ci

bspc: $(MAPCHECK) $(RENDER) $(BSPINFO) $(CI)

build: check-map $(SDHLT)/sdHLRAD $(CI)
	timeout $(TIMEOUT) $(PY) maps/$(MAP)/build.py
	@echo "compiling $(MAP) with SDHLT ($(LIGHT) lighting, $(VIS) vis)"
	$(call compile,$(BD),$(MAP))
	$(CI) package $(MAP)

test: check-map $(MAPCHECK)
	$(MAPCHECK) structure $(BD)/$(MAP).bsp
	$(MAPCHECK) placements $(BD)/$(MAP).bsp
	$(MAPCHECK) wiring $(BD)/$(MAP).bsp
	@if [ -f maps/$(MAP)/tests/routes.py ]; then \
	  ROUTES_JSON=1 $(PY) maps/$(MAP)/tests/routes.py > $(BD)/routes.json && \
	  timeout 900 $(MAPCHECK) routes $(BD)/$(MAP).bsp $(BD)/routes.json; fi

views: check-map $(RENDER)
	$(PY) -m tools.render.views $(MAP) > $(BD)/views.json
	timeout 900 $(RENDER) views $(BD)/$(MAP).bsp $(BD)/valve $(BD)/views.json $(BD)/previews

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

readme: $(CI)
	$(CI) readme $(if $(V),--version $(V))

# The pinned SDHLT build (tools/bsp/sdhlt.rev) must keep compiling the template map to the same bytes, so a new
# SDHLT commit, a different build of it, or a change to the template's generator shows up as a deliberate
# `make golden-update` in the PR diff. Only the small template map: SDHLT's lighting floats differ in the last
# bits between C++ compilers and CPUs on bigger maps.
GOLDEN_MAPS = _template
GOLDEN_BSP  = tools/bsp/golden.sha256
GO_OUT      = build/golden

golden-out: $(SDHLT)/sdHLRAD
	@mkdir -p $(GO_OUT); rm -f $(GO_OUT)/*
	@set -e; for m in $(GOLDEN_MAPS); do \
	  echo "golden: $$m"; \
	  timeout 900 $(PY) maps/$$m/build.py > /dev/null; \
	  (cd build/$$m && timeout 900 $(SDHLT)/sdHLCSG -nowadtextures $$m && timeout 900 $(SDHLT)/sdHLBSP $(BSPFLAGS) $$m \
	    && timeout 900 $(SDHLT)/sdHLVIS -fast $$m && timeout 900 $(SDHLT)/sdHLRAD -fast -bounce 2 $$m) > /dev/null; \
	  cp build/$$m/$$m.bsp $(GO_OUT)/$$m.bsp; \
	  (cd $(GO_OUT) && sha256sum $$m.bsp) >> $(GO_OUT)/golden.sha256; \
	done

golden: golden-out
	@diff -u $(GOLDEN_BSP) $(GO_OUT)/golden.sha256 && echo "compiler golden: identical" || \
	  (echo "compiler output changed. If that is intended, run make golden-update and commit the result."; exit 1)

golden-update: golden-out
	cp $(GO_OUT)/golden.sha256 $(GOLDEN_BSP)
	@echo "updated $(GOLDEN_BSP); commit it and say in the PR why the output changed"

# golangci-lint (config: tools/.golangci.yml) covers gofmt, go vet, staticcheck, errcheck, unused and ineffassign.
# Install v2.15.0, the version CI pins: https://golangci-lint.run/docs/welcome/install/local/
lint:
	@if command -v golangci-lint >/dev/null; then timeout 600 golangci-lint run -c tools/.golangci.yml ./...; else \
	  echo "golangci-lint not installed (CI runs it); falling back to gofmt + go vet"; \
	  test -z "$$(gofmt -l tools)" || (echo "gofmt needed:"; gofmt -l tools; exit 1); \
	  go vet ./...; fi
