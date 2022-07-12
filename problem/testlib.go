package problem

import (
	"os"
	"regexp"
	"rindag/etc"
	"strconv"

	"github.com/criyle/go-judge/pb"
	log "github.com/sirupsen/logrus"
)

const MaxTestlibMessageLen = 1024

var (
	TestlibFile *pb.Request_File
)

func ParseTestlibOutput(output string, fullScore int64) (
	pb.Response_Result_StatusType, int64, string) {
	accepted := func() (pb.Response_Result_StatusType, func(string) string) {
		return pb.Response_Result_Accepted, func(s string) string {
			if len(s) <= MaxTestlibMessageLen-3 {
				return "AC " + s
			}
			return "AC " + s[:MaxTestlibMessageLen-6] + "..."
		}
	}
	wrongAnswer := func() (pb.Response_Result_StatusType, func(string) string) {
		return pb.Response_Result_WrongAnswer, func(s string) string {
			if len(s) <= MaxTestlibMessageLen-3 {
				return "WA " + s
			}
			return "WA " + s[:MaxTestlibMessageLen-6] + "..."
		}
	}
	formatError := func() (pb.Response_Result_StatusType, func(string) string) {
		return pb.Response_Result_WrongAnswer, func(s string) string {
			if len(s) <= MaxTestlibMessageLen-3 {
				return "PE " + s
			}
			return "PE " + s[:MaxTestlibMessageLen-6] + "..."
		}
	}
	partiallyCorrect := func() (pb.Response_Result_StatusType, func(string) string) {
		return pb.Response_Result_PartiallyCorrect, func(s string) string {
			if len(s) <= MaxTestlibMessageLen-3 {
				return "PC " + s
			}
			return "PC " + s[:MaxTestlibMessageLen-6] + "..."
		}
	}

	status := pb.Response_Result_JudgementFailed
	score := int64(0)
	message := output
	builder := func(s string) string {
		if len(s) <= MaxTestlibMessageLen {
			return s
		}
		return s[:MaxTestlibMessageLen-3] + "..."
	}
	if result := regexp.MustCompile(`^ok (.*)$`).FindStringSubmatch(output); result != nil {
		status, builder = accepted()
		score = fullScore
		message = result[1]
	} else if result :=
		regexp.MustCompile(`^wrong answer (.*)$`).FindStringSubmatch(output); result != nil {
		status, builder = wrongAnswer()
		message = result[1]
	} else if result :=
		regexp.MustCompile(`^wrong output format (.*)$`).FindStringSubmatch(output); result != nil {
		status, builder = formatError()
		message = result[1]
	} else if result :=
		regexp.MustCompile(`^(?:partially correct|points) \(?([0-9.]*)\)? (.*)$`).
			FindStringSubmatch(output); result != nil {
		p, _ := strconv.ParseFloat(result[1], 64)
		if p >= 1 {
			status, builder = accepted()
			score = fullScore
			message = result[2]
		} else if p > 0 {
			status, builder = partiallyCorrect()
			score = int64(float64(fullScore) * p)
			message = result[1] + " " + result[2]
		} else {
			status, builder = wrongAnswer()
			message = result[2]
		}
	}
	return status, score, builder(message)
}

func init() {
	// Read testlib code from config
	path := etc.Config.Testlib.Path
	if _, err := os.Stat(etc.Config.Testlib.Path); err != nil {
		log.WithError(err).Fatal("Failed to read testlib code")
	}
	TestlibFile = &pb.Request_File{
		File: &pb.Request_File_Local{Local: &pb.Request_LocalFile{Src: path}}}
	log.Info("Read testlib code")
}
