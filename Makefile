.PHONY: build
build: build-configure-codeql build-verify-scans build-emass-promotion

.PHONY: build-configure-codeql
build-configure-codeql:
	echo "Building configure-codeql"
	go build -o bin/configure-codeql ./configure-codeql/cmd

.PHONY: build-emass-promotion
build-emass-promotion:
	echo "Building emass-promotion"
	go build -o bin/emass-promotion ./emass-promotion/cmd

.PHONY: build-verify-scans
build-verify-scans:
	echo "Building verify-scans"
	go build -o bin/verify-scans ./verify-scans/cmd

.PHONY: docker
docker: docker-build docker-push

.PHONY: docker-build
docker-build: docker-build-configure-codeql docker-build-verify-scans docker-build-emass-promotion

.PHONY: docker-push
docker-push: docker-push-configure-codeql docker-push-verify-scans docker-push-emass-promotion

.PHONY: docker-run
docker-run: docker-run-configure-codeql docker-run-verify-scans docker-run-emass-promotion

# Configure CodeQL
.PHONY: docker-build-configure-codeql
docker-build-configure-codeql:
	echo "Building docker image for configure-codeql"
	docker build -t ghcr.io/department-of-veterans-affairs/codeql-tools:configure-codeql -f configure-codeql/Dockerfile .

.PHONY: docker-push-configure-codeql
docker-push-configure-codeql:
	echo "Pushing docker image for configure-codeql"
	docker push ghcr.io/department-of-veterans-affairs/codeql-tools:configure-codeql

.PHONY: docker-run-configure-codeql
docker-run-configure-codeql:
	echo "Running docker image for configure-codeql"
	./scripts/docker-run-configure-codeql.sh

#eMASS Promotion
.PHONY: docker-build-emass-promotion
docker-build-emass-promotion:
	echo "Building docker image for emass-promotion"
	docker build -t ghcr.io/department-of-veterans-affairs/codeql-tools:emass-promotion -f emass-promotion/Dockerfile .

.PHONY: docker-push-emass-promotion
docker-push-emass-promotion:
	echo "Pushing docker image for emass-promotion"
	docker push ghcr.io/department-of-veterans-affairs/codeql-tools:emass-promotion

.PHONY: docker-run-emass-promotion
docker-run-emass-promotion:
	echo "Running docker image for emass-promotion"
	./scripts/docker-run-emass-promotion.sh

# Verify Scans
.PHONY: docker-build-verify-scans
docker-build-verify-scans:
	echo "Building docker image for verify-scans"
	docker build -t ghcr.io/department-of-veterans-affairs/codeql-tools:verify-scans -f verify-scans/Dockerfile .

.PHONY: docker-push-verify-scans
docker-push-verify-scans:
	echo "Pushing docker image for verify-scans"
	docker push ghcr.io/department-of-veterans-affairs/codeql-tools:verify-scans

.PHONY: docker-run-verify-scans
docker-run-verify-scans:
	echo "Running docker image for verify-scans"
	./scripts/docker-run-verify-scans.sh

.PHONY: npm-build-parse-build-steps
npm-build-parse-build-steps:
	echo "Building docker image for parse-build-steps"
	docker build -t ghcr.io/department-of-veterans-affairs/codeql-tools:parse-build-steps -f parse-build-steps/Dockerfile .

.PHONY: deps
deps:
	go get -u ./...
	go mod tidy
	go mod vendor
