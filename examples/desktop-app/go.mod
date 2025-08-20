module github.com/scttfrdmn/aws-remote-access-patterns/examples/desktop-app

go 1.21

require (
	github.com/aws/aws-sdk-go-v2 v1.24.0
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.141.0
	github.com/aws/aws-sdk-go-v2/service/s3 v1.47.5
	github.com/aws/aws-sdk-go-v2/service/sts v1.26.5
	github.com/pkg/browser v0.0.0-20210911075715-681adbf594b8
	github.com/scttfrdmn/aws-remote-access-patterns v0.0.0
)

replace github.com/scttfrdmn/aws-remote-access-patterns => ../../