module github.com/GoogleCloudPlatform/terraformer

go 1.25.0

require (
	cloud.google.com/go v0.123.0 // indirect
	cloud.google.com/go/logging v1.19.1
	cloud.google.com/go/storage v1.64.0
	github.com/Azure/azure-sdk-for-go v68.0.0+incompatible
	github.com/Azure/azure-storage-blob-go v0.10.0
	github.com/Azure/go-autorest/autorest v0.11.30
	github.com/aws/aws-sdk-go-v2 v1.43.5
	github.com/aws/aws-sdk-go-v2/config v1.32.36
	github.com/aws/aws-sdk-go-v2/credentials v1.19.35
	github.com/aws/aws-sdk-go-v2/service/accessanalyzer v1.51.5
	github.com/aws/aws-sdk-go-v2/service/acm v1.43.5
	github.com/aws/aws-sdk-go-v2/service/apigateway v1.42.5
	github.com/aws/aws-sdk-go-v2/service/appsync v1.56.5
	github.com/aws/aws-sdk-go-v2/service/autoscaling v1.71.1
	github.com/aws/aws-sdk-go-v2/service/batch v1.68.5
	github.com/aws/aws-sdk-go-v2/service/budgets v1.46.5
	github.com/aws/aws-sdk-go-v2/service/cloud9 v1.36.5
	github.com/aws/aws-sdk-go-v2/service/cloudformation v1.76.2
	github.com/aws/aws-sdk-go-v2/service/cloudfront v1.67.5
	github.com/aws/aws-sdk-go-v2/service/cloudhsmv2 v1.37.5
	github.com/aws/aws-sdk-go-v2/service/cloudtrail v1.58.5
	github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.66.4
	github.com/aws/aws-sdk-go-v2/service/cloudwatchevents v1.35.5
	github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs v1.82.1
	github.com/aws/aws-sdk-go-v2/service/codebuild v1.72.5
	github.com/aws/aws-sdk-go-v2/service/codecommit v1.36.5
	github.com/aws/aws-sdk-go-v2/service/codedeploy v1.38.5
	github.com/aws/aws-sdk-go-v2/service/codepipeline v1.49.5
	github.com/aws/aws-sdk-go-v2/service/cognitoidentity v1.36.5
	github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider v1.67.5
	github.com/aws/aws-sdk-go-v2/service/configservice v1.68.5
	github.com/aws/aws-sdk-go-v2/service/datapipeline v1.33.5
	github.com/aws/aws-sdk-go-v2/service/devicefarm v1.42.1
	github.com/aws/aws-sdk-go-v2/service/docdb v1.51.5
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.63.2
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.321.1
	github.com/aws/aws-sdk-go-v2/service/ecr v1.60.5
	github.com/aws/aws-sdk-go-v2/service/ecrpublic v1.41.5
	github.com/aws/aws-sdk-go-v2/service/ecs v1.90.1
	github.com/aws/aws-sdk-go-v2/service/efs v1.44.5
	github.com/aws/aws-sdk-go-v2/service/eks v1.90.5
	github.com/aws/aws-sdk-go-v2/service/elasticache v1.56.5
	github.com/aws/aws-sdk-go-v2/service/elasticbeanstalk v1.37.5
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing v1.36.5
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2 v1.58.6
	github.com/aws/aws-sdk-go-v2/service/elasticsearchservice v1.45.5
	github.com/aws/aws-sdk-go-v2/service/emr v1.64.5
	github.com/aws/aws-sdk-go-v2/service/firehose v1.46.5
	github.com/aws/aws-sdk-go-v2/service/glue v1.152.1
	github.com/aws/aws-sdk-go-v2/service/iam v1.58.2
	github.com/aws/aws-sdk-go-v2/service/identitystore v1.39.5
	github.com/aws/aws-sdk-go-v2/service/iot v1.77.5
	github.com/aws/aws-sdk-go-v2/service/kafka v1.58.1
	github.com/aws/aws-sdk-go-v2/service/kinesis v1.46.5
	github.com/aws/aws-sdk-go-v2/service/kms v1.55.5
	github.com/aws/aws-sdk-go-v2/service/lambda v1.101.3
	github.com/aws/aws-sdk-go-v2/service/medialive v1.101.4
	github.com/aws/aws-sdk-go-v2/service/mediapackage v1.42.5
	github.com/aws/aws-sdk-go-v2/service/mediastore v1.32.5
	github.com/aws/aws-sdk-go-v2/service/mq v1.39.5
	github.com/aws/aws-sdk-go-v2/service/opsworks v1.31.0
	github.com/aws/aws-sdk-go-v2/service/organizations v1.53.6
	github.com/aws/aws-sdk-go-v2/service/qldb v1.32.2
	github.com/aws/aws-sdk-go-v2/service/rds v1.124.2
	github.com/aws/aws-sdk-go-v2/service/redshift v1.65.5
	github.com/aws/aws-sdk-go-v2/service/resourcegroups v1.36.5
	github.com/aws/aws-sdk-go-v2/service/route53 v1.65.7
	github.com/aws/aws-sdk-go-v2/service/s3 v1.107.1
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.44.5
	github.com/aws/aws-sdk-go-v2/service/securityhub v1.76.1
	github.com/aws/aws-sdk-go-v2/service/servicecatalog v1.42.5
	github.com/aws/aws-sdk-go-v2/service/ses v1.37.5
	github.com/aws/aws-sdk-go-v2/service/sfn v1.45.5
	github.com/aws/aws-sdk-go-v2/service/sns v1.42.5
	github.com/aws/aws-sdk-go-v2/service/sqs v1.46.5
	github.com/aws/aws-sdk-go-v2/service/ssm v1.73.5
	github.com/aws/aws-sdk-go-v2/service/ssoadmin v1.43.2
	github.com/aws/aws-sdk-go-v2/service/sts v1.45.5
	github.com/aws/aws-sdk-go-v2/service/swf v1.37.5
	github.com/aws/aws-sdk-go-v2/service/waf v1.33.5
	github.com/aws/aws-sdk-go-v2/service/wafregional v1.33.5
	github.com/aws/aws-sdk-go-v2/service/wafv2 v1.77.4
	github.com/aws/aws-sdk-go-v2/service/workspaces v1.73.2
	github.com/aws/aws-sdk-go-v2/service/xray v1.39.5
	github.com/hashicorp/go-azure-helpers v0.36.0
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-hclog v1.6.3
	github.com/hashicorp/go-plugin v1.8.0
	github.com/hashicorp/hcl v1.0.0
	github.com/hashicorp/terraform-plugin-go v0.31.0
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.10.2
	github.com/spf13/pflag v1.0.10
	github.com/zclconf/go-cty v1.19.0
	golang.org/x/oauth2 v0.36.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	gonum.org/v1/gonum v0.17.0
	google.golang.org/api v0.292.0
	google.golang.org/genproto v0.0.0-20260810153831-ec0a7760b754
)

require github.com/gofrs/uuid v3.2.0+incompatible // indirect

require (
	github.com/Azure/azure-pipeline-go v0.2.3 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.24 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.1 // indirect
	github.com/Azure/go-autorest/autorest/to v0.4.0 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.3.1 // indirect
	github.com/Azure/go-autorest/logger v0.2.2 // indirect
	github.com/Azure/go-autorest/tracing v0.6.1 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.17 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.36 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.36 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.36 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.37 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.16 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.9.29 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.12.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.36 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.37 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.33.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.38.5 // indirect
	github.com/aws/smithy-go v1.27.7
	github.com/fatih/color v1.19.0 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/googleapis/gax-go/v2 v2.23.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.8 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/hashicorp/go-version v1.9.0 // indirect
	github.com/hashicorp/yamux v0.1.2 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/mattn/go-colorable v0.1.15 // indirect
	github.com/mattn/go-ieproxy v0.0.1 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/oklog/run v1.1.0 // indirect
	golang.org/x/crypto v0.54.0 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	google.golang.org/grpc v1.83.0
	google.golang.org/protobuf v1.36.12
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

require (
	cloud.google.com/go/cloudbuild v1.32.0
	cloud.google.com/go/cloudtasks v1.19.0
	cloud.google.com/go/iam v1.13.0
	cloud.google.com/go/monitoring v1.30.0
	github.com/manicminer/hamilton v0.44.0
)

require (
	cel.dev/expr v0.25.2 // indirect
	cloud.google.com/go/auth v0.22.0 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	cloud.google.com/go/longrunning v1.2.0 // indirect
	github.com/Azure/go-autorest/autorest/azure/cli v0.4.4 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp v1.33.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric v0.57.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/internal/resourcemapping v0.57.0 // indirect
	github.com/apparentlymart/go-textseg/v15 v15.0.0 // indirect
	github.com/apparentlymart/go-textseg/v17 v17.0.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.5.5 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cncf/xds/go v0.0.0-20260202195803-dba9d589def2 // indirect
	github.com/dimchansky/utfbom v1.1.1 // indirect
	github.com/envoyproxy/go-control-plane/envoy v1.37.0 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.3.3 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-jose/go-jose/v4 v4.1.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.2 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.20 // indirect
	github.com/manicminer/hamilton-autorest v0.2.0 // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 // indirect
	github.com/spiffe/go-spiffe/v2 v2.7.0 // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/detectors/gcp v1.44.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.68.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.67.0 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/sdk v1.44.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260810153831-ec0a7760b754 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260810153831-ec0a7760b754 // indirect
)

require (
	github.com/aws/aws-sdk-go v1.44.334
	github.com/aws/aws-sdk-go-v2/service/directconnect v1.44.2
)

replace gopkg.in/jarcoal/httpmock.v1 => github.com/jarcoal/httpmock v1.0.5

replace gopkg.in/ns1/ns1-go.v2 => github.com/ns1/ns1-go/v2 v2.6.5

replace github.com/tencentcloud/tencentcloud-sdk-go => github.com/tencentcloud/tencentcloud-sdk-go v1.0.392
