package middlewares

import (
	"context"
	"net/http"
	"time"

	"github.impcloud.net/Responsive-Retail-MVP/rfid-alert-service/pkg/web"

	log "github.com/sirupsen/logrus"
)

// Logger middleware
func Logger(next web.Handler) web.Handler {
	return web.Handler(func(ctx context.Context, writer http.ResponseWriter, request *http.Request) error {

		tracerID := ctx.Value(web.KeyValues).(*web.ContextValues).TraceID
		start := time.Now()
		err := next(ctx, writer, request)

		if request.URL.EscapedPath() != "/" {
			log.WithFields(log.Fields{
				"Method":     request.Method,
				"RequestURI": request.RequestURI,
				"Duration":   time.Since(start),
				"TracerId":   tracerID,
			}).Debug("Http Logger middleware")
		}
		// return err since it will contain the error or nil
		return err
	})
}
