package handler

import (
	"context"

	"rindag/service/judge"
)

// GetJudgeFile reads a file from the judge.
func GetJudgeFile(ctx context.Context, judgeID string, fileID string) ([]byte, error) {
	j, err := judge.GetJudge(judgeID)
	if err != nil {
		return nil, err
	}
	file, err := j.GetFile(ctx, fileID)
	if err != nil {
		return nil, err
	}
	return file.Content, nil
}
