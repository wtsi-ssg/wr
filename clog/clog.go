/*******************************************************************************
 * Copyright (c) 2020 Genome Research Ltd.
 *
 * Author: Sendu Bala <sb10@sanger.ac.uk>
 * Based on: https://blog.gopheracademy.com/advent-2016/context-logging/
 *
 * Permission is hereby granted, free of charge, to any person obtaining
 * a copy of this software and associated documentation files (the
 * "Software"), to deal in the Software without restriction, including
 * without limitation the rights to use, copy, modify, merge, publish,
 * distribute, sublicense, and/or sell copies of the Software, and to
 * permit persons to whom the Software is furnished to do so, subject to
 * the following conditions:
 *
 * The above copyright notice and this permission notice shall be included
 * in all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
 * EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
 * MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
 * IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
 * CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
 * TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
 * SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
 ******************************************************************************/

// package clog is used to do contextual logging with a global logger.
package clog

import (
	"bytes"
	"context"

	log "github.com/inconshreveable/log15"
	"github.com/sb10/l15h"
)

// init sets our default logging syle.
func init() {
	ToDefault()
}

// ToDefault sets the global logger to log to STDERR at the "warn" level.
func ToDefault() {
	toOutputAtLevel(log.StderrHandler, log.LvlWarn)
}

// toOutputAtLevel sets the handler of the global logger to filter on the given
// level, add caller info, and output to the given handler.
func toOutputAtLevel(outputHandler log.Handler, lvl log.Lvl) {
	h := log.LvlFilterHandler(
		lvl,
		l15h.CallerInfoHandler(
			outputHandler,
		),
	)
	log.Root().SetHandler(h)
}

// ToBufferAtLevel sets the global logger to log to the returned
// bytes.Buffer at the given level.
func ToBufferAtLevel(lvl string) *bytes.Buffer {
	buff := new(bytes.Buffer)
	toOutputAtLevel(log.StreamHandler(buff, log.LogfmtFormat()), lvlFromString(lvl))

	return buff
}

// ToFileAtLevel sets the global logger to log to a file at the given path
// and at the given level.
func ToFileAtLevel(path, lvl string) error {
	fh, err := log.FileHandler(path, log.LogfmtFormat())
	if err != nil {
		return err
	}

	toOutputAtLevel(fh, lvlFromString(lvl))

	return nil
}

// lvlFromString returns a log.Lvl for the given string. Valid lvls are
// "debug"|"dbug", "info", "warn", "error"|"eror", "crit". Invalid lvls return
// LvlDebug.
func lvlFromString(lvl string) log.Lvl {
	logLevel, err := log.LvlFromString(lvl)
	if err != nil {
		return log.LvlDebug
	}

	return logLevel
}

// logger returns the global logger with as much context as possible.
func logger(ctx context.Context) log.Logger {
	logger := log.Root()
	if ctx != nil {
		logger = addStringKeyToLogger(ctx, logger, retrySetKey, "retryset")
		logger = addStringKeyToLogger(ctx, logger, retryActivityKey, "retryactivity")
		logger = addIntKeyToLogger(ctx, logger, retryNumKey, "retrynum")
	}

	return logger
}

// addStringKeyToLogger checks if the given string key is set in the logger and
// returns a new logger with that context under the logger key if so.
func addStringKeyToLogger(ctx context.Context, logger log.Logger, key correlationIDType, loggerKey string) log.Logger {
	if val, ok := ctx.Value(key).(string); ok {
		logger = logger.New(loggerKey, val)
	}

	return logger
}

// addIntKeyToLogger checks if the given int key is set in the logger and
// returns a new logger with that context under the logger key if so.
func addIntKeyToLogger(ctx context.Context, logger log.Logger, key correlationIDType, loggerKey string) log.Logger {
	if val, ok := ctx.Value(key).(int); ok {
		logger = logger.New(loggerKey, val)
	}

	return logger
}

// Debug logs the given message with context and args to the global logger at
// the debug level. Caller info is included.
func Debug(ctx context.Context, msg string, args ...interface{}) {
	logger(ctx).Debug(msg, args...)
}

// Info logs the given message with context and args to the global logger at
// the info level.
func Info(ctx context.Context, msg string, args ...interface{}) {
	logger(ctx).Info(msg, args...)
}

// Warn logs the given message with context and args to the global logger at
// the warn level. Caller info is included.
func Warn(ctx context.Context, msg string, args ...interface{}) {
	logger(ctx).Warn(msg, args...)
}

// Error logs the given message with context and args to the global logger at
// the error level. Caller info is included.
func Error(ctx context.Context, msg string, args ...interface{}) {
	logger(ctx).Error(msg, args...)
}

// Crit logs the given message with context and args to the global logger at
// the error level. A stack trace is included.
func Crit(ctx context.Context, msg string, args ...interface{}) {
	logger(ctx).Crit(msg, args...)
}
