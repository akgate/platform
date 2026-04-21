package logging

import "log/slog"

type Slogger struct {
	logger *slog.Logger
}

func NewSlogger(logger *slog.Logger) Logger {
	return &Slogger{logger: logger}
}

func (l *Slogger) Debug(msg string, fields ...Field) {
	l.logger.Debug(msg, l.toArgs(fields...)...)
}

func (l *Slogger) Info(msg string, fields ...Field) {
	l.logger.Info(msg, l.toArgs(fields...)...)
}

func (l *Slogger) Warn(msg string, fields ...Field) {
	l.logger.Warn(msg, l.toArgs(fields...)...)
}

func (l *Slogger) Error(msg string, fields ...Field) {
	l.logger.Error(msg, l.toArgs(fields...)...)
}

func (l *Slogger) With(fields ...Field) Logger {
	return &Slogger{
		logger: l.logger.With(l.toArgs(fields...)...),
	}
}

func (l *Slogger) toArgs(fields ...Field) []any {
	args := make([]any, 0, len(fields)*2)
	for _, f := range fields {
		args = append(args, f.Key, f.Value)
	}
	return args
}
