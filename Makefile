.PHONY: server web test packbuild

server:
	go run ./cmd/server

web:
	cd web && npm run dev

test:
	go test $$(go list ./... | grep -v /node_modules/)
	cd web && npm test --if-present

packbuild:
	go run ./cmd/packbuild
