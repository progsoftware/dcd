package dcd_test

import (
	"context"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/smithy-go"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/progsoftware/dcd/internal/dcd"
)

var dbClient *dynamodb.Client

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Define the DynamoDB container request
	req := testcontainers.ContainerRequest{
		Image:        "amazon/dynamodb-local:latest",
		ExposedPorts: []string{"8000/tcp"},
		WaitingFor:   wait.ForListeningPort("8000/tcp"),
	}

	// Start the container
	dynamoDBContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		log.Fatalf("Failed to start the DynamoDB container: %s", err)
	}

	defer func() {
		if err := dynamoDBContainer.Terminate(ctx); err != nil {
			log.Fatalf("Could not stop dynamodb container: %s", err)
		}
	}()

	// Get the container's mapped port
	mappedPort, err := dynamoDBContainer.MappedPort(ctx, "8000")
	if err != nil {
		log.Fatalf("Failed to get the container's mapped port: %s", err)
	}

	host, err := dynamoDBContainer.Host(ctx)
	if err != nil {
		log.Fatalf("Failed to get the container's host: %s", err)
	}

	awsConfig, err := config.LoadDefaultConfig(ctx, config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
		func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL: "http://" + host + ":" + mappedPort.Port(),
				//SigningRegion: "us-west-2",
			}, nil
		},
	)))
	awsConfig.Credentials = credentials.NewStaticCredentialsProvider("test", "test", "test")
	if err != nil {
		log.Fatalf("Failed to load AWS config: %s", err)
	}

	// Create the DynamoDB client
	dbClient = dynamodb.NewFromConfig(awsConfig)

	// Create the test table
	createTableInput := &dynamodb.CreateTableInput{
		TableName: aws.String("test-table"),
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("PK"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("PK"),
				KeyType:       types.KeyTypeHash,
			},
		},
		BillingMode: "PAY_PER_REQUEST",
	}
	if _, err := dbClient.CreateTable(ctx, createTableInput); err != nil {
		log.Fatalf("Failed to create table: %s", err)
	}

	// Run the tests
	code := m.Run()

	os.Exit(code)
}

func TestMissingTableError(t *testing.T) {
	ctx := context.Background()

	awsBackend := dcd.NewAWSBackend(dbClient, "missing-table")

	_, err := awsBackend.GetBuildID(ctx)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	// check error is of type types.ResourceNotFoundException
	smithyErr, ok := err.(*smithy.OperationError)
	if !ok {
		t.Fatalf("Expected *smithy.OperationError, got %T", err)
	}
	if !strings.Contains(smithyErr.Err.Error(), "ResourceNotFoundException") {
		t.Fatalf("Expected error to contain 'ResourceNotFoundException', got %s", smithyErr.Err.Error())
	}
}

func TestIncrementBuildID(t *testing.T) {
	ctx := context.Background()

	awsBackend := dcd.NewAWSBackend(dbClient, "test-table")

	// First call, item does not exist
	id, err := awsBackend.GetBuildID(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if id != 1 {
		t.Errorf("Expected ID to be 1, got %d", id)
	}

	// Second call, item exists
	id, err = awsBackend.GetBuildID(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if id != 2 {
		t.Errorf("Expected ID to be 2, got %d", id)
	}
}

func TestStartPipeline(t *testing.T) {
	// Given
	pipeline := dcd.NewPipeline()
	pipeline.SetMetadata(&dcd.Metadata{
		Component: "test-component",
		GitSHA:    "test-git-sha",
	})
	pipeline.SetDefinition(&dcd.PipelineDefinition{
		Steps: []dcd.Step{},
	})

	// When
	eventsChan, err := pipeline.Run()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Then
	var events []dcd.Event
	for event := range eventsChan {
		events = append(events, event)
	}

	//buildID := events[0].(dcd.PipelineStartEvent).BuildID

}
