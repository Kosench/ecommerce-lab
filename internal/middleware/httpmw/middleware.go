package httpmw

import (
	"net/http"
	"time"

	"github.com/Kosench/ecommerce-lab/platform/logger"
	"go.uber.org/zap"
)

func Logging(next http.Handler, logger logger.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Логируем начало запроса (только для долгих операций в продакшене)
		// В разработке логируем все
		logger.Debug("request started",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("remote_addr", r.RemoteAddr),
		)

		// Оборачиваем ResponseWriter чтобы получить статус ответа
		ww := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Выполняем следующий обработчик
		next.ServeHTTP(ww, r)

		duration := time.Since(start)

		// Логируем завершение запроса
		// 5xx — error, 4xx — warn, остальное — info
		logFields := []zap.Field{
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", ww.statusCode),
			zap.Duration("duration", duration),
			zap.String("remote_addr", r.RemoteAddr),
		}

		switch {
		case ww.statusCode >= 500:
			logger.Error("request failed",
				append(logFields, zap.Error(ww.err))...,
			)
		case ww.statusCode >= 400:
			logger.Warn("request warning",
				logFields...,
			)
		default:
			logger.Info("request completed",
				logFields...,
			)
		}
	})
}

// PanicRecoveryMiddleware ловит паники и возвращает 500
func Recovery(next http.Handler, logger logger.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.Error("request panicked",
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.String("remote_addr", r.RemoteAddr),
					zap.Any("recovered", rec),
					zap.Stack("stack"),
				)

				http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	err        error
	written    bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.statusCode = http.StatusOK
		rw.written = true
	}
	return rw.ResponseWriter.Write(b)
}
