module github.com/scttfrdmn/aws-remote-access-patterns/examples/aws-cli-helper

go 1.21

require (
	github.com/aws/aws-sdk-go-v2 v1.24.0
	github.com/aws/aws-sdk-go-v2/config v1.26.1
	github.com/aws/aws-sdk-go-v2/credentials v1.16.12
	github.com/aws/aws-sdk-go-v2/service/sts v1.26.5
	github.com/scttfrdmn/aws-remote-access-patterns v0.0.0
	golang.org/x/crypto v0.17.0
	gopkg.in/yaml.v3 v3.0.1
)

replace github.com/scttfrdmn/aws-remote-access-patterns => ../../