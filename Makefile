.PHONY: docker
docker: docker-verify-scans

.PHONY: docker-push
docker-push: docker-verify-scans-push

.PHONY: docker-verify-scans
docker-verify-scans:
	echo "Building docker image for verify-scans"
	docker build -t ghcr.io/department-of-veterans-affairs/codeql-tools:verify-scans -f verify-scans/Dockerfile .

.PHONY: docker-verify-scans-push
docker-verify-scans-push:
	echo "Pushing docker image for verify-scans"
	docker push ghcr.io/department-of-veterans-affairs/codeql-tools:verify-scans