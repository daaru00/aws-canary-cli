## About this project

Right now the only methods available to deploy AWS Canaries are: 
- Using CloudFormation template and place the script code inline in a string inside a yml file...
- Using the web console (until the page is reloaded) and don't worry about code versioning, pipelines and those other complex things...

This is a third method that allows you to version the code, deploy and manage canaries. 
When you don't want them anymore use the remove and related resources will be also removed (like Lambda Function and Layer Versions).

PS: Also tried [Serverless Components](https://github.com/daaru00/serverless-component-synthetics-canary) and I had some deployment problems.

## Install CLI

Download last archive package version from [releases page](https://github.com/daaru00/aws-canary-cli/releases):

* Windows: aws-canary_VERSION_Windows_x86_64.zip
* Mac: aws-canary_VERSION_Darwin_x86_64.tar.gz
* Linux: aws-canary_VERSION_Linux_x86_64.tar.gz

Unpack it and copy `aws-canary` into one of your executable paths, for example, for Mac and Linux users:
```bash
tar -czvf aws-canary_*.tar.gz
sudo mv aws-canary /usr/local/bin/aws-canary
rm aws-canary_*.tar.gz
```

### For Linux Users

You can also install CLI from deb or rpm package downloading from releases page:

* aws-canary_1.0.0_linux_amd64.deb
* aws-canary_1.0.0_linux_amd64.rpm

### For Mac Users

Unlock CLI executable file going to "System Preference > Security and Privacy > General" and click on button "open anyway".

## Commands

Usage:
```bash
./aws-canary [global options] command [command options] [path...]
```

- **deploy**: Deploy a Synthetics Canary
- **remove**: Remove a Synthetics Canary
- **start**: Start a Synthetics Canary
- **stop**: Stop a Synthetics Canary
- **logs**: Return Synthetics Canary Run logs
- **results**: Return Synthetics Canary Runs
- **help**: Shows a list of commands or help for one command

## Environment configuration file

This CLI also load environment variable from `.env` file in current working directory:
```
AWS_PROFILE=my-profile
AWS_REGION=us-east-1

CANARY_ARTIFACT_BUCKET_NAME=my-bucket-bucket-name
``` 

Setting `CANARY_ENV` environment variable is it possible to load different env file:
```bash
export CANARY_ENV=""
aws-canary deploy # will load .env file
```
```bash
export CANARY_ENV="prod"
aws-canary deploy # will load .env.prod file
```
```bash
export CANARY_ENV="STAGE"
aws-canary deploy # will load .env.STAGE file
```

## Canary configuration file

This CLI will search for `canary.yml` configurations files, recursively, in search path (provided via first argument of any commands) for configurations file and deploy/remove canaries in parallels. The canary configuration file looks like this:
```yaml
name: test         # canary name
memory: 1000       # minimum required memory, in MB
timeout: 840       # maximum timeout (14 minutes), in seconds
tracing: false     # enable active tracing
env:                      # canary environment variables
  ENDPOINT: "https://example.com"
  PAGE_LOAD_TIMEOUT: 15000
retention:
  failure: 31              # retention for failure results, in days
  success: 31              # retention for success results, in days
schedule:
  duration: 0                 # run only once when it starts, or regular run in period (in seconds)
  expression: "rate(0 hour)"  # run only manually with 0 value or rate(30 minutes)
tags:                         # canary tags
  Project: test
  Environment: test
```

### Code

Code path and handler can be changed from defaults:
```yaml
name: test
code:
  handler: index.handler  # this value must end with the string ".handler"
  src: ./                 # relative to config file
  exclude:                # path excluded from zip archive
    - ".git/**"
    - ".gitignore"
    - ".DS_Store"
    - "npm-debug.log"
    - "canary.yml"
```

### Interpolation

In configuration file it is possible to interpolate environment variables using `${var}` or `$var` syntax:
```yaml
name: test
env:
  ENDPOINT: "${ENDPOINT_FROM_ENV}"
tags:
  Project: "${APP_NAME}"
  Environment: "${ENV}"
```

### Schedule

Schedule configurations for running only manually (when executing start command):
```yaml
schedule:
  duration: 0
  expression: "rate(0 hour)"
```
or with a schedule expression:
```yaml
schedule:
  duration: 0
  expression: "rate(5 minutes)"
```
canary will not start automatically until you manually run it with start command (or adding `--start` to deploy one).

For more information refer to AWS doc about [AWS::Synthetics::Canary Schedule property](https://docs.aws.amazon.com/it_it/AWSCloudFormation/latest/UserGuide/aws-properties-synthetics-canary-schedule.html).

### VPC

Optionally Canary can run in a VPC:
```yaml
name: test
env:
  ENDPOINT: "${ENDPOINT_FROM_ENV}"
vpc:
  subnets:
    - subnet-xxxxxxx # VPC subnets ids (at least one)
    - subnet-xxxxxxx
    - subnet-xxxxxxx
  securityGroups:
    - sg-xxxxxxxxx   # VPC security group ids
```

### Custom policy

Canary can have a custom role attached to it:
```yaml
name: test
role: my-role-name  # use an existing, custom IAM role
```
otherwise, custom policy statement can be directly attached to the role:
```yaml
name: test
policies:           # policies statement to attach to IAM role. If the role property is set, this property is ignored. 
  - Effect: "Allow"
    Action: 
      - "dynamodb:ListTables"
    Resource: 
      - "*"
```

Custom policy statement must respect a strict format:
```
Effect: String
Action: Array of strings
Resource: Array of strings
Condition:
  StringEquals: Map of string[string]
```

Policies entries defined in policies are merged with the default provided by the cli:
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "xray:PutTraceSegments"
      ],
      "Resource": [
        "*"
      ]
    },
    {
      "Effect": "Allow",
      "Action": [
        "ssm:GetParameter*",
      ],
      "Resource": [
        "arn:aws:ssm:us-east-1:1234567890:parameter/cwsyn/*"
      ]
    },
    {
      "Effect": "Allow",
      "Action": [
        "cloudwatch:PutMetricData"
      ],
      "Resource": [
        "*"
      ],
      "Condition": {
        "StringEquals": {
          "cloudwatch:namespace": "CloudWatchSynthetics"
        }
      }
    },
    {
      "Effect": "Allow",
      "Action": [
        "logs:CreateLogStream",
        "logs:PutLogEvents",
        "logs:CreateLogGroup"
      ],
      "Resource": [
        "arn:aws:logs:us-east-1:1234567890:log-group:/aws/lambda/cwsyn-*"
      ]
    },
    {
      "Effect": "Allow",
      "Action": [
        "s3:PutObject"
      ],
      "Resource": [
        "arn:aws:s3:::<artifact bucket name>/*"
      ]
    },
    {
      "Effect": "Allow",
      "Action": [
        "s3:GetBucketLocation"
      ],
      "Resource": [
        "arn:aws:s3:::<artifact bucket name>"
      ]
    },
    {
      "Effect": "Allow",
      "Action": [
        "s3:ListAllMyBuckets"
      ],
      "Resource": [
        "*"
      ]
    },
    {
      "Effect": "Allow",
      "Action": [
        "ec2:CreateNetworkInterface",
				"ec2:DescribeNetworkInterface",
				"ec2:DescribeNetworkInterfaces",
				"ec2:DeleteNetworkInterface"
      ],
      "Resource": [
        "*"
      ]
    }
  ]
}
```
It's the original AWS one with some additions: 
- SSM parameters read-only access for paths that starts with `/cwsyn/`.
- EC2 network interface CRUD access in order to be able to use Canary in a VPC.

### Search path

Any command accept file or directory paths as arguments, any canary configuration file that match will be loaded an added to list.

If a directory is provided the CLI will search recursively for files `canary.yml` (configurable via `--config-config-file`) 
and try to parse them using YAML parser (configurable via `--config-config-parser`), for example:
```bash
aws-canary deploy ./examples
```

Search path and config file name can be set via environment variable (or `.env` file):
```
CANARY_PATH=./tests/e2e/
CANARY_CONFIG_FILE=*.yml
```

If a file is provided the CLI will be try to parse using YAML parser (configurable via `--config-config-parser`), for example:
```bash
aws-canary deploy examples/nodejs/simple/canary.yml
```

Search path can be multiple, every argument respect the rules mentioned above:
```bash
aws-canary deploy examples/nodejs/simple/canary.yml examples/nodejs/web/canary.yml examples/python/simple/canary.yml
# load 3 canaries from provided files

aws-canary deploy examples/nodejs/ examples/python/simple/canary.yml
# load all canaries in nodejs directory and a single one from python

aws-canary deploy examples/nodejs/ examples/python/
# load all canaries from nodejs and python directories (all)
```

Also a file glob pattern can be used as search paths:
```bash
aws-canary deploy examples/**/simple/canary.yml
# load 2 canaries, one in nodejs directory and the other in the python one
```

Here an example of project configuration with single canary:
```bash
.
└── canary
    ├── canary.yml
    └── index.js
```

Here an example of project configuration with multiple canaries:
```bash
.
└── canaries
    ├── cart
    │   ├── canary.yml
    │   └── index.js
    ├── home
    │   ├── canary.yml
    │   └── index.js
    └── login
        ├── canary.yml
        └── index.js
```

Configuration file name can be changed via `--config-file` parameter:
```bash
aws-canary deploy --config-file="test.yml" /tests/e2e/
.
└── tests
    └── e2e
        ├── cart
        │   ├── test.yml
        │   └── index.js
        ├── home
        │   ├── test.yml
        │   └── index.js
        └── login
            ├── test.yml
            └── index.js
```
both exact match and wildcard pattern can be used:
```bash
aws-canary deploy --config-file="*.yml" /tests/e2e/
.
└── tests
    └── e2e
        ├── cart.yml
        ├── home.yml
        ├── login.yml
        └── index.js # export multiple handlers
```

## Build canaries code

An command `build` is provided in order to install dependencies for canaries that need to, so this command is not required if you don't use npm or pip dependencies.

Build code (install dependencies)
```bash
aws-canary build
```

Adding `--output` flag the build process wil print the output at the end of command:
```bash
aws-canary build --output
```
will print an output similar to this:
```
[test-js-deps] Output: 
audited 1 package in 1.087s
found 0 vulnerabilities
```

If there are no `package.json` or `requirements.txt` files in canary directory, no commands will run.

## Deploy canaries

To deploy canaries run the `deploy` command:
```bash
aws-canary deploy
```

Artifact bucket name can be customized using `--artifact-bucket` parameter:
```bash
aws-canary deploy --artifact-bucket my-bucket-bucket-name
```

Adding `--yes` flag the deploy process wil automatically create artifact and source bucket (if needed) required for canary execution:
```bash
aws-canary deploy --artifact-bucket my-bucket-bucket-name --yes
```

If the artifact size exceed the maximum that Lambda allow (512 KB), an error wil be returned:
```
AWSLambdaException: 
        status code: 500, request id: xxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxx
exit status 1
```
using `--upload` flag the zip code will be uploaded to the source bucket instead of directly passing it during canary creation/update:
```
aws-canary deploy --upload
```

Source bucket name can be customized using `--sources-bucket` parameter:
```bash
aws-canary deploy --sources-bucket my-sources-bucket-name --upload
```

## Start canaries (manually execution)

To state canaries manually run the `start` command:
```bash
aws-canary start
```

## Stop canaries (manually execution)

To state canaries manually run the `stop` command:
```bash
aws-canary stop
```

## Retrieve canaries logs

To retrieve canary runs' logs run the `logs` command:
```bash
aws-canary logs
```

To retrieve last canary run's logs run the `logs` command with `--last` flag:
```bash
aws-canary logs --last
```

## Retrieve canaries results

To retrieve canary runs' results run the `results` command:
```bash
aws-canary results
```

To retrieve last canary run's results run the `results` command with `--last` flag:
```bash
aws-canary results --last
```

## Remove canaries

To remove (only) canaries run the `remove` command:
```bash
aws-canary remove
```
The related Lambda function and Layer Versions (with name that starts with "cwsyn-") are also cleaned

In order to also remove artifact and source bucket with canaries run the `remove` command with `--delete-artifact-bucket` or `--delete-sources-bucket` flag:
```bash
aws-canary remove --delete-artifact-bucket --artifact-bucket my-bucket-bucket-name
# delete only artifact bucket

aws-canary remove --delete-sources-bucket --sources-bucket my-sources-bucket-name
# delete only source bucket

aws-canary remove --delete-sources-bucket --delete-artifact-bucket --sources-bucket my-sources-bucket-name --artifact-bucket my-bucket-bucket-name 
# delete both
```
