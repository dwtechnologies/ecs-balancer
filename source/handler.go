package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

type cloudwatchEventDetail struct {
	ClusterArn           string `json:"clusterArn"`
	ContainerInstanceArn string `json:"containerInstanceArn"`
	AgentConnected       bool   `json:"agentConnected"`
}

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, event events.CloudWatchEvent) error {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	eventDetail := &cloudwatchEventDetail{}

	err := json.Unmarshal(event.Detail, eventDetail)
	if err != nil {
		log.Println("error unmarshaling event")
		return err
	}

	client, err := newECSClient()
	if err != nil {
		log.Printf("can't get ECS client %s", err)
		return err
	}

	// describe the container instance that triggered the event
	r, err := client.DescribeContainerInstancesRequest(&ecs.DescribeContainerInstancesInput{
		Cluster:            aws.String(eventDetail.ClusterArn),
		ContainerInstances: []string{eventDetail.ContainerInstanceArn},
	}).Send(ctx)
	if err != nil {
		return err
	}

	// "Container Instance State Change Events" (https://docs.aws.amazon.com/AmazonECS/latest/developerguide/ecs_cwe_events.html)
	// covers a lot of scenario. We assume that:
	// "The Amazon ECS container agent registers a container instance for the first time"
	//
	// check nr of container instances
	if len(r.ContainerInstances) == 0 {
		log.Printf("no container instances available for %s", eventDetail.ContainerInstanceArn)
		return nil
	}

	// nr of tasks (running, pending)
	log.Printf("%s Status %s\n", eventDetail.ContainerInstanceArn, *r.ContainerInstances[0].Status)
	log.Printf("%s RunningTasksCount %d\n", eventDetail.ContainerInstanceArn, *r.ContainerInstances[0].RunningTasksCount)
	log.Printf("%s PendingTasksCount %d\n", eventDetail.ContainerInstanceArn, *r.ContainerInstances[0].PendingTasksCount)

	// rebalance cluster if no tasks are running/pending
	if *r.ContainerInstances[0].Status == "ACTIVE" &&
		*r.ContainerInstances[0].RunningTasksCount == 0 &&
		*r.ContainerInstances[0].PendingTasksCount == 0 &&
		*r.ContainerInstances[0].AgentConnected {

		err := balanceCluster(ctx, client, eventDetail.ClusterArn)
		if err != nil {
			return err
		}
	}

	return nil
}

func newECSClient() (*ecs.Client, error) {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return nil, err
	}

	return ecs.New(cfg), err
}

func balanceCluster(ctx context.Context, client *ecs.Client, cluster string) error {
	// list services
	r, err := client.ListServicesRequest(&ecs.ListServicesInput{
		Cluster:    aws.String(cluster),
		LaunchType: ecs.LaunchTypeEc2,
	}).Send(ctx)
	if err != nil {
		return err
	}

	// describe services
	rr, err := client.DescribeServicesRequest(&ecs.DescribeServicesInput{
		Cluster:  aws.String(cluster),
		Services: r.ServiceArns,
	}).Send(ctx)

	for _, s := range rr.Services {
		if *s.DesiredCount > 1 {
			log.Printf("balancing service: %s (%d instances)", *s.ServiceName, *s.DesiredCount)

			// force new deployment
			_, err := client.UpdateServiceRequest(&ecs.UpdateServiceInput{
				Cluster:            aws.String(cluster),
				Service:            s.ServiceName,
				ForceNewDeployment: aws.Bool(true),
			}).Send(ctx)
			if err != nil {
				return err
			}
		}
	}

	return err
}
