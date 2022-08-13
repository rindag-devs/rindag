package problem

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"rindag/service/storage"

	"github.com/minio/minio-go/v7"
	"gopkg.in/yaml.v3"
)

var packageFuncs = map[string]func(*Problem, map[string]*TestGroup, io.Writer) error{
	"luogu": LuoguPackager,
}

type luoguProblemConfig struct {
	TimeLimit   int64  `yaml:"timeLimit"`
	MemoryLimit uint64 `yaml:"memoryLimit"`
	Score       int32  `yaml:"score"`
	SubtaskID   int    `yaml:"subtaskId"`
}

func LuoguPackager(p *Problem, testGroups map[string]*TestGroup, out io.Writer) error {
	ctx := context.Background()
	bucket, err := p.Bucket()
	if err != nil {
		return err
	}

	packW := zip.NewWriter(out)
	defer packW.Close()

	data, err := packW.Create("data.zip")
	if err != nil {
		return err
	}

	dataW := zip.NewWriter(data)

	writeToZip := func(path string) error {
		obj, err := storage.Client.GetObject(ctx, bucket, path, minio.GetObjectOptions{})
		if err != nil {
			return err
		}

		fw, err := dataW.Create(path)
		if err != nil {
			return err
		}

		if _, err := io.Copy(fw, obj); err != nil {
			return err
		}

		return nil
	}

	groupID := 0
	conf := make(map[string]luoguProblemConfig)
	scoB := strings.Builder{}

	scoB.WriteString("@total_score = 0\n")
	scoB.WriteString("@final_status = AC\n")
	scoB.WriteString("@final_time = 0\n")
	scoB.WriteString("@final_memory = 0\n")

	for _, group := range testGroups {
		groupID++

		scoB.WriteString(fmt.Sprintf("@total_score = @total_score + @score%d\n", groupID))
		scoB.WriteString(fmt.Sprintf(
			"if @time%d < @final_time; then\n  @final_time = @time%d\nfi\n", groupID, groupID))
		scoB.WriteString(fmt.Sprintf(
			"if @memory%d < @final_memory; then\n  @final_memory = @memory%d\nfi\n", groupID, groupID))
		scoB.WriteString(fmt.Sprintf(
			"if @status%d != AC and (@final_status == AC or @status%d != UNAC); then\n"+
				"  @final_status = @status%d\n"+
				"fi\n",
			groupID, groupID, groupID))

		for _, test := range group.Tests {
			infPath := test.Prefix + ".in"
			ansPath := test.Prefix + ".ans"

			if err := writeToZip(infPath); err != nil {
				return err
			}

			if err := writeToZip(ansPath); err != nil {
				return err
			}

			conf[infPath] = luoguProblemConfig{
				TimeLimit:   time.Duration(group.TimeLimit).Milliseconds(),
				MemoryLimit: group.MemoryLimit / (1024 * 1024),
				Score:       group.FullScore,
				SubtaskID:   groupID,
			}
		}
	}

	confD, err := yaml.Marshal(conf)
	if err != nil {
		return err
	}

	confW, err := dataW.Create("config.yml")
	if err != nil {
		return err
	}

	if _, err := confW.Write(confD); err != nil {
		return err
	}

	if err := dataW.Close(); err != nil {
		return err
	}

	scoW, err := packW.Create("scoring.txt")
	if err != nil {
		return err
	}

	if _, err := scoW.Write([]byte(scoB.String())); err != nil {
		return err
	}

	return nil
}

// Package is a function to make a package.
func (p *Problem) Package(
	format string, lang string, testGroups map[string]*TestGroup, out io.Writer,
) error {
	if _, ok := packageFuncs[format]; !ok {
		return fmt.Errorf("unknown package format: %s", format)
	}

	return packageFuncs[format](p, testGroups, out)
}
