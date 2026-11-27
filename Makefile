.PHONY: server web test packbuild packbuild-broad packbuild-krillion

server:
	go run ./cmd/server

web:
	cd web && npm run dev

test:
	go test $$(go list ./... | grep -v /node_modules/)
	cd web && npm test --if-present

packbuild:
	go run ./cmd/packbuild --defs packs/defs --out packs/core-v1.json

packbuild-broad:
	go run ./cmd/packbuild --defs packs/defs-broad --out packs/broad-v1.json

# Fetch Krillion reveals (CEO-approved) into packs/import/krillion and build packs/krillion-v1.json
packbuild-krillion:
	go run ./cmd/packbuild --fetch-krillion --krillion-import packs/import/krillion --out packs/krillion-v1.json
