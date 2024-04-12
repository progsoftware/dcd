package dcd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type AWSBackend struct {
	tableName string
	dynamodb  *dynamodb.Client
}

func NewAWSBackend(dynamodb *dynamodb.Client, tableName string) *AWSBackend {
	return &AWSBackend{
		tableName: tableName,
		dynamodb:  dynamodb,
	}
}

func (b *AWSBackend) GetBuildID(ctx context.Context) (int64, error) {
	key := map[string]types.AttributeValue{
		"PK": &types.AttributeValueMemberS{Value: "BUILD_ID"},
	}

	update := &dynamodb.UpdateItemInput{
		TableName:        aws.String(b.tableName),
		Key:              key,
		UpdateExpression: aws.String("SET #id = if_not_exists(#id, :start) + :inc"),
		ExpressionAttributeNames: map[string]string{
			"#id": "ID",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":inc":   &types.AttributeValueMemberN{Value: "1"},
			":start": &types.AttributeValueMemberN{Value: "0"},
		},
		ReturnValues: types.ReturnValueUpdatedNew,
	}

	output, err := b.dynamodb.UpdateItem(ctx, update)
	if err != nil {
		return 0, err
	}

	newID, ok := output.Attributes["ID"].(*types.AttributeValueMemberN)
	if !ok {
		return 0, fmt.Errorf("unexpected type for ID")
	}

	id, err := strconv.ParseInt(newID.Value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse new ID: %w", err)
	}

	return id, nil
}

func (b *AWSBackend) StartPipeline(ctx context.Context, buildID string) error {
	panic("TODO")
}

func (b *AWSBackend) PutPipeline(ctx context.Context, state *PipelineState) error {
	panic("TODO")
}

func (b *AWSBackend) PutPipelineEvent(ctx context.Context, event Event) error {
	panic("TODO")
}
