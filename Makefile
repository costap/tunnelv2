
generate:
	@buf generate

cert:
	@cd scripts/certs; ./gen.sh; cd ../..
	@cp scripts/certs/server-cert.pem cert/server-cert.pem
	@cp scripts/certs/server-key.pem cert/server-key.pem
	@cp scripts/certs/ca-cert.pem cert/ca-cert.pem

.PHONY: cert