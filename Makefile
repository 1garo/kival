unit_tests:
	go test -v ./...

.PHONY: integration_tests
integration_tests:
	@./scripts/integration_tests.sh $(filter-out $@,$(MAKECMDGOALS))

%:
	@:
