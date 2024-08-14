package utils

import (
	"context"
	"github.com/go-logr/logr"
	"k8s.io/klog/v2"
	"strings"
)

func Log(args ...any) {
	args[0] = "===== 🤖 " + args[0].(string)
	klog.Info(args...)
}

func Logf(format string, args ...any) {
	klog.Infof("===== 🤖 "+format, args...)
}

func LogErr(format string, args ...any) {
	klog.Infof("===== 😵 "+format, args...)
}

func NewKLog(ctx context.Context) logr.Logger {
	return klog.FromContext(ctx)
}

func StringP(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}
