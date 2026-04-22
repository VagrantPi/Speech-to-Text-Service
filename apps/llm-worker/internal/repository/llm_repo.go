package repository

import "context"

type LLMRepo interface {
	GenerateSummaryStream(ctx context.Context, transcript string, tokenChan chan<- string) (fullSummary string, err error)
}
