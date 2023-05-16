.PHONY: docker
docker: docker-build docker-push

.PHONY: docker-build
docker-build: docker-build-configure-codeql docker-build-verify-scans

.PHONY: docker-push
docker-push: docker-push-configure-codeql docker-push-verify-scans

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

.PHONY: deps
deps:
	go get -u ./...
	go mod tidy
	go mod vendor