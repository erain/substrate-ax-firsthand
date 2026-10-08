.PHONY: test sources bootstrap prepare install smoke cleanup

test:
	go test -race ./...
	go vet ./...
	@for script in scripts/*.sh scripts/ate scripts/ax scripts/kube scripts/router scripts/request scripts/dispatch; do bash -n "$$script"; done
	@for script in scripts/ate scripts/ax scripts/kube scripts/router scripts/request scripts/dispatch; do sh -n "$$script"; done
	bash scripts/test-guards.sh

sources:
	bash scripts/sources.sh

bootstrap:
	bash scripts/bootstrap.sh

prepare:
	bash scripts/prepare.sh

install:
	bash scripts/install.sh

smoke:
	bash scripts/smoke.sh

cleanup:
	bash scripts/cleanup.sh
