
ENVIRONMENT  ?= prod
AWS_REGION   ?= eu-west-1
AWS_PROFILE  ?=
PROJECT      ?= itops
OWNER        ?= cloudops
SERVICE_NAME ?= ecs-balancer
S3_BUCKET    ?=
ECS_CLUSTER  ?= default

###

deploy: build package-cf deploy-cf

build:
	@docker run --rm \
		-v $(PWD)/source:/src \
		-w /src \
		golang:stretch sh -c \
			'apt-get update && apt-get install -y zip && \
			echo "\n▸ building code..." && \
			cd /src/ && go test -v -cover && go build -o main && \
			zip handler.zip main && \
			rm main && \
			echo "▸ build done..."'

build-native:
	cd source; GOOS=linux go test -v -cover && go build -o main && zip handler.zip main

package-cf:
	mkdir -p build
	aws cloudformation package \
		--template-file template.yaml \
		--output-template-file build/template.yaml \
		--s3-bucket $(S3_BUCKET) \
		--s3-prefix $(PROJECT)/$(SERVICE_NAME)

deploy-cf:
	aws cloudformation deploy \
		--template-file build/template.yaml \
		--stack-name ecs-balancer-$(ECS_CLUSTER) \
		--parameter-overrides \
			ecsCluster=$(ECS_CLUSTER) \
		--tags \
                        Environment=$(ENVIRONMENT) \
                        Project=$(PROJECT) \
                        Owner=$(OWNER) \
                --capabilities CAPABILITY_IAM
	rm -rf build source/main source/handler.zip

clean:
	rm -rf build source/main source/handler.zip

# eof
