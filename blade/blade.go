package blade

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/TylerBrock/colorjson"
	"github.com/TylerBrock/saw/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/fatih/color"
)

// A Blade is a Saw execution instance
type Blade struct {
	config *config.Configuration
	aws    *config.AWSConfiguration
	output *config.OutputConfiguration
	cwl    *cloudwatchlogs.Client
}

// NewBlade creates a new Blade with CloudWatchLogs instance from provided config
func NewBlade(
	config *config.Configuration,
	awsConfig *config.AWSConfiguration,
	outputConfig *config.OutputConfiguration,
) (*Blade, error) {
	cfg, err := awsConfig.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	blade := &Blade{
		cwl:    cloudwatchlogs.NewFromConfig(cfg),
		config: config,
		output: outputConfig,
	}

	return blade, nil
}

// GetLogGroups gets the log groups from AWS given the blade configuration
func (b *Blade) GetLogGroups(ctx context.Context) []types.LogGroup {
	input := b.config.DescribeLogGroupsInput()
	groups := make([]types.LogGroup, 0)

	paginator := cloudwatchlogs.NewDescribeLogGroupsPaginator(b.cwl, input)
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			fmt.Println("Error", err)
			os.Exit(2)
		}
		groups = append(groups, output.LogGroups...)
	}
	return groups
}

// GetLogStreams gets the log streams from AWS given the blade configuration
func (b *Blade) GetLogStreams(ctx context.Context) []types.LogStream {
	input := b.config.DescribeLogStreamsInput()
	streams := make([]types.LogStream, 0)

	paginator := cloudwatchlogs.NewDescribeLogStreamsPaginator(b.cwl, input)
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			fmt.Println("Error", err)
			os.Exit(2)
		}
		streams = append(streams, output.LogStreams...)
	}
	return streams
}

// GetEvents gets events from AWS given the blade configuration
func (b *Blade) GetEvents(ctx context.Context) {
	formatter := b.output.Formatter()
	input := b.config.FilterLogEventsInput()

	paginator := cloudwatchlogs.NewFilterLogEventsPaginator(b.cwl, input)
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			fmt.Println("Error", err)
			os.Exit(2)
		}

		for _, event := range output.Events {
			if b.output.Pretty {
				fmt.Println(formatEvent(formatter, event))
			} else {
				fmt.Println(aws.ToString(event.Message))
			}
		}
	}
}

// StreamEvents continuously prints log events to the console
func (b *Blade) StreamEvents(ctx context.Context) {
	var lastSeenTime *int64
	seenEventIDs := make(map[string]bool)
	formatter := b.output.Formatter()
	input := b.config.FilterLogEventsInput()

	clearSeenEventIds := func() {
		seenEventIDs = make(map[string]bool)
	}

	for {
		paginator := cloudwatchlogs.NewFilterLogEventsPaginator(b.cwl, input)
		for paginator.HasMorePages() {
			output, err := paginator.NextPage(ctx)
			if err != nil {
				fmt.Println("Error", err)
				os.Exit(2)
			}

			for _, event := range output.Events {
				timestamp := aws.ToInt64(event.Timestamp)
				if lastSeenTime == nil || timestamp > *lastSeenTime {
					lastSeenTime = &timestamp
					clearSeenEventIds()
				}

				eventID := aws.ToString(event.EventId)
				if !seenEventIDs[eventID] {
					var message string
					if b.output.Raw {
						message = aws.ToString(event.Message)
					} else {
						message = formatEvent(formatter, event)
					}
					message = strings.TrimRight(message, "\n")
					fmt.Println(message)
					seenEventIDs[eventID] = true
				}
			}
		}

		if lastSeenTime != nil {
			input.StartTime = lastSeenTime
		}
		time.Sleep(1 * time.Second)
	}
}

// formatEvent returns a CloudWatch log event as a formatted string using the provided formatter
func formatEvent(formatter *colorjson.Formatter, event types.FilteredLogEvent) string {
	red := color.New(color.FgRed).SprintFunc()
	white := color.New(color.FgWhite).SprintFunc()

	str := aws.ToString(event.Message)
	bytes := []byte(str)
	date := time.Unix(0, aws.ToInt64(event.Timestamp)*int64(time.Millisecond))
	dateStr := date.Format(time.RFC3339)
	streamStr := aws.ToString(event.LogStreamName)
	jl := map[string]interface{}{}

	if err := json.Unmarshal(bytes, &jl); err != nil {
		return fmt.Sprintf("[%s] (%s) %s", red(dateStr), white(streamStr), str)
	}

	output, _ := formatter.Marshal(jl)
	return fmt.Sprintf("[%s] (%s) %s", red(dateStr), white(streamStr), output)
}
