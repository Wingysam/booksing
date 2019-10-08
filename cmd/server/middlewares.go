package main

import (
	"math"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gnur/booksing"
	"github.com/sirupsen/logrus"
)

var timeFormat = time.RFC3339

// Logger is the logrus logger handler
func Logger(log *logrus.Entry) gin.HandlerFunc {

	return func(c *gin.Context) {
		// other handler can change c.Path so:
		path := c.Request.URL.Path
		start := time.Now()
		c.Next()
		stop := time.Since(start)
		latency := int(math.Ceil(float64(stop.Nanoseconds()) / 1000000.0))
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		clientUserAgent := c.Request.UserAgent()
		referer := c.Request.Referer()
		dataLength := c.Writer.Size()
		if dataLength < 0 {
			dataLength = 0
		}

		entry := log.WithFields(logrus.Fields{
			"statusCode": statusCode,
			"latency":    latency, // time to process
			"clientIP":   clientIP,
			"method":     c.Request.Method,
			"path":       path,
			"referer":    referer,
			"dataLength": dataLength,
			"userAgent":  clientUserAgent,
		})

		if len(c.Errors) > 0 {
			entry.Error(c.Errors.ByType(gin.ErrorTypePrivate).String())
		} else {
			msg := "GIN handled request"
			if statusCode > 499 {
				entry.Error(msg)
			} else if statusCode > 399 {
				entry.Warn(msg)
			} else {
				entry.Info(msg)
			}
		}
	}
}

func (app *booksingApp) APIKeyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		apikey := c.GetHeader("x-api-key")
		if apikey == "" {
			c.JSON(403, gin.H{
				"msg": "access denied",
			})
			c.Abort()
			return
		}
		key, err := app.db.GetAPIKey(apikey)
		if err != nil {
			c.JSON(500, gin.H{
				"msg": err.Error(),
			})
			c.Abort()
			return
		}
		key.LastUsed = time.Now().In(app.timezone)
		app.db.SaveAPIKey(key)

		c.Set("apikey", key.ID)
		c.Set("id", &booksing.User{
			Username: key.Username,
		})

	}
}

func (app *booksingApp) BearerTokenMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		rawHeader := c.GetHeader("Authorization")
		app.logger.WithField("header", rawHeader).Info("got header")
		if rawHeader == "" {
			app.logger.Warning("No auth header provided")
			c.JSON(403, gin.H{
				"msg": "access denied",
			})
			c.Abort()
			return
		}
		if !strings.HasPrefix(rawHeader, "Bearer ") {
			app.logger.Warning("Incorrect formatted header")
			c.JSON(403, gin.H{
				"msg": "access denied",
			})
			c.Abort()
			return
		}

		jwt := strings.TrimPrefix(rawHeader, "Bearer ")

		token, err := app.validateToken(jwt)
		if err != nil {
			app.logger.WithField("err", err).Error("could not validate provided bearer token")
			c.JSON(403, gin.H{
				"msg": "access denied",
			})
			c.Abort()
			return
		}

		user, ok := token.Claims["email"]
		if !ok {
			app.logger.Error("could not extract email from claims")
			c.JSON(403, gin.H{
				"msg": "access denied",
			})
			c.Abort()
			return
		}

		username, ok := user.(string)
		if !ok {
			app.logger.Error("could not cast user to string")
			c.JSON(403, gin.H{
				"msg": "access denied",
			})
			c.Abort()
			return
		}

		u, err := app.db.GetUser(username)
		if err == booksing.ErrNotFound {
			err = app.db.SaveUser(&booksing.User{
				Username: username,
				Created:  time.Now(),
				LastSeen: time.Now(),
			})
			if err != nil {
				app.logger.WithField("err", err).Error("could not save new user")
				c.JSON(500, gin.H{
					"msg": "internal server error",
				})
				c.Abort()
				return
			}
		} else if err == nil {
			u.LastSeen = time.Now()
			err = app.db.SaveUser(&u)
			if err != nil {
				app.logger.Error("could not update user")
				c.JSON(500, gin.H{
					"msg": "internal server error",
				})
				c.Abort()
				return
			}
		} else {
			app.logger.WithField("err", err).Error("could not get user")
			c.JSON(500, gin.H{
				"msg": "internal server error",
			})
			c.Abort()
			return
		}

		c.Set("id", &booksing.User{
			Username: username,
		})

	}
}

func (app *booksingApp) CronMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		rawHeader := c.GetHeader("X-Appengine-Cron")
		if rawHeader != "true" {
			app.logger.Warning("cron called without valid header")
			c.JSON(403, gin.H{
				"msg": "access denied",
			})
			c.Abort()
			return
		}

	}
}