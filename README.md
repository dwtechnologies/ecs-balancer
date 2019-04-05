# ⚖️  ecs-balancer

When a new ECS instance registers in a cluster, ECS doesn't have a way to reschedule tasks in order to fill the new capacity or spread the worload among AZs.

This Lambda function is triggered by "Container Instance State Change Events" CloudWatch event and:
- check nr of running and pending tasks for ContainerInstanceArn
- if both values are 0 and ECS agent is connected, trigger a cluster balance by upating all the services with 'DesiredCount > 1'

### Service resource (placement strategies)
```yaml
  Service:
    Type: "AWS::ECS::Service"
    Properties:
      PlacementStrategies:
        - Type: spread
          Field: attribute:ecs.availability-zone
        - Type: spread
          Field: instanceId
```

### CloudWatchEvent

```json
{
  "version": "0",
  "id": "8952ba83-7be2-4ab5-9c32-6687532d15a2",
  "detail-type": "ECS Container Instance State Change",
  "source": "aws.ecs",
  "account": "111122223333",
  "time": "2016-12-06T16:41:06Z",
  "region": "eu-west-1",
  "resources": [
    "arn:aws:ecs:eu-west-1:111122223333:container-instance/b54a2a04-046f-4331-9d74-3f6d7f6ca315"
  ],
  "detail": {
    "agentConnected": true,
    "attributes": [],
    "clusterArn": "arn:aws:ecs:eu-west-1:111122223333:cluster/default",
    "containerInstanceArn": "arn:aws:ecs:eu-west-1:111122223333:container-instance/803b97fd-1da1-4e31-8080-6b5b8f34123e",
    "ec2InstanceId": " i-00043580a9a035076"
  }
}
```


### Deployment requirements
- docker
- make
- aws cli


### Deployment
Use the included `Makefile` to deploy the resources.

The `OWNER` env var is for tagging. So you can set this to what you want.
The `ENVIRONMENT` env var is also for naming + tagging, but will also be included in CloudWatch logs.
This so you can make out differences between dev, test and prod etc. if you're running them on the same AWS Account.

```bash
AWS_PROFILE=my-profile AWS_REGION=region OWNER=TeamName S3_BUCKET=my-artifact-bucket ECS_CLUSTER=target-ecs-cluster make deploy
```

Example
```bash
AWS_PROFILE=default AWS_REGION=eu-west-1 OWNER=cloudops S3_BUCKET=my-artifact-bucket ECS_CLUSTER=cluster-one-prod make deploy
```

------

**Note**: according to [AWS documentation](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/ecs_cwe_events.html), this CW event covers a lot of scenarios.
To avoid too many invokes, CW event pattern is restricted to a specific ECS cluster:

```yaml
Pattern:
  source: ["aws.ec2"]
  detail-type: ["EC2 Spot Instance Interruption Warning"]
  detail:
    clusterArn: 
      - !Sub "arn:aws:ecs:${AWS::Region}:${AWS::AccountId}:cluster/${clusterName}"
```

