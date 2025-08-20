module github.com/scttfrdmn/aws-remote-access-patterns/examples/cli-tool

go 1.21

require (
	github.com/aws/aws-sdk-go-v2 v1.24.0
	github.com/aws/aws-sdk-go-v2/service/sts v1.26.5
	github.com/fatih/color v1.16.0
	github.com/scttfrdmn/aws-remote-access-patterns v0.0.0
	github.com/spf13/cobra v1.8.0
	github.com/spf13/viper v1.18.2
	gopkg.in/ini.v1 v1.67.0
)

require (
	github.com/aws/aws-sdk-go-v2/config v1.26.1
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.141.0
	github.com/aws/aws-sdk-go-v2/service/s3 v1.47.5
)

replace github.com/scttfrdmn/aws-remote-access-patterns => ../../