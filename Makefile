unit:
	go test -v ./...

.PHONY: integration_tests
e2e:
	@./scripts/integration_tests.sh $(filter-out $@,$(MAKECMDGOALS))

%:
	@:
