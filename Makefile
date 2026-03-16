.PHONY: build
build:
	go build -o browsh ./web

.PHONY: ui
ui:
	cd web/ui && bun run build
