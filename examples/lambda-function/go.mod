module github.com/scttfrdmn/aws-remote-access-patterns/examples/lambda-function

go 1.21

require (
	github.com/aws/aws-lambda-go v1.41.0
	github.com/aws/aws-sdk-go-v2 v1.24.0
	github.com/aws/aws-sdk-go-v2/config v1.26.1
	github.com/aws/aws-sdk-go-v2/service/s3 v1.47.5
	github.com/aws/aws-sdk-go-v2/service/sts v1.26.5
	github.com/scttfrdmn/aws-remote-access-patterns v0.0.0
)

replace github.com/scttfrdmn/aws-remote-access-patterns => ../../