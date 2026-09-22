VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TS ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
MODULE   ?= $(shell grep '^module ' go.mod | awk '{print $$2}')
GROUP_ID ?=
PKG      := $(MODULE)/internal/app
LDFLAGS  := -X '$(PKG).Version=$(VERSION)' -X '$(PKG).BuildTime=$(BUILD_TS)' -X '$(PKG).ModuleName=$(MODULE)'

.PHONY: all build test lint cyclo fmt mocks swagger licenses coverage coverage-html coverage-report clean docker i18n certs

all: fmt build test lint cyclo

build:
	go build -ldflags "$(LDFLAGS)" -o oauth2-auth-code-proxy .

test:
	go test -race -coverprofile=coverage.out -covermode=atomic -coverpkg=./... -count=1 ./...

lint:
	addlicense -check -f .copyright.tmpl -ignore "**/*.json" -ignore "**/*.yaml" -ignore "**/*.yml" -ignore "**/*.md" -ignore "**/testdata/**" -ignore "certs/**" -ignore ".target/**" -ignore "vendor/**" .
	golangci-lint run

cyclo:
	gocyclo -over 9 ./internal/

swagger:
	swag init --v3.1 --generalInfo doc.go --dir ./internal --output .target/swagger --outputTypes json --parseDependency --parseInternal
	cp .target/swagger/swagger.json internal/docs/swagger.json
	cp .target/swagger/swagger.json api/swagger.json

mocks:
	mockery

fmt:
	addlicense -f .copyright.tmpl -ignore "**/*.json" -ignore "**/*.yaml" -ignore "**/*.yml" -ignore "**/*.md" -ignore "**/testdata/**" -ignore "certs/**" -ignore ".target/**" -ignore "vendor/**" .
	goimports -w . && go fmt ./...

coverage: test
	go tool cover -func=coverage.out

coverage-html: test
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

coverage-report: test
	@go tool cover -func=coverage.out | tee coverage-report.txt
	@echo ""
	@echo "Summary:"
	@go tool cover -func=coverage.out | grep total:

docker: certs
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIMESTAMP=$(BUILD_TS) \
		--build-arg GROUP_ID=$(GROUP_ID) \
		-t ghcr.io/mathworks-ref-arch/container-images/oauth2-auth-code-proxy:$(VERSION) \
		-t ghcr.io/mathworks-ref-arch/container-images/oauth2-auth-code-proxy:latest .

licenses:
	@go-licenses report . --include_tests 2>/dev/null | grep -v "$(shell go list -m)" | sort > /tmp/licenses.csv
	@echo "# Third-Party Software Notices" > THIRD_PARTY_NOTICES.md
	@echo "" >> THIRD_PARTY_NOTICES.md
	@echo "This software includes the following third-party components." >> THIRD_PARTY_NOTICES.md
	@echo "" >> THIRD_PARTY_NOTICES.md
	@echo "| Module | License | Source |" >> THIRD_PARTY_NOTICES.md
	@echo "|--------|---------|--------|" >> THIRD_PARTY_NOTICES.md
	@awk -F',' '{printf "| %s | %s | [source](%s) |\n", $$1, $$3, $$2}' /tmp/licenses.csv >> THIRD_PARTY_NOTICES.md
	@rm -f /tmp/licenses.csv

certs: certs/tls.crt

certs/tls.crt:
	@mkdir -p certs
	MSYS_NO_PATHCONV=1 openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:prime256v1 \
		-keyout certs/tls.key -out certs/tls.crt -days 3650 -nodes \
		-subj "/CN=oauth2-auth-code-proxy" \
		-addext "subjectAltName=DNS:localhost,IP:127.0.0.1"

i18n:
	cd internal && go generate -run gotext

clean:
	rm -f oauth2-auth-code-proxy coverage.out coverage.html coverage-report.txt
